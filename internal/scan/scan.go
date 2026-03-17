package scan

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/packages"

	"github.com/shuymn/exportsurf/pkg/report"
)

type Options struct {
	Patterns             []string
	WorkingDir           string
	TreatTestsAsExternal bool
	ExcludePackages      []string
	ExcludeSymbols       []string
	IncludeMethods       bool
	IncludeFields        bool
}

type candidateState struct {
	candidate      report.Candidate
	externalRefPkg map[string]struct{}
}

type fileInfo struct {
	name      string
	isTest    bool
	generated bool
}

func Run(opts Options) ([]report.Candidate, error) {
	if len(opts.Patterns) == 0 {
		return nil, errors.New("at least one pattern is required")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir:   opts.WorkingDir,
		Tests: true,
	}

	pkgs, err := packages.Load(cfg, opts.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, errors.New("package loading failed")
	}

	var ifaces []namedInterface
	if opts.IncludeMethods {
		ifaces = collectInterfaces(pkgs)
	}

	var fkeys *fieldKeyMap
	if opts.IncludeFields {
		fkeys = buildFieldKeyMap(pkgs)
	}

	states := collectDefinitions(pkgs, opts.WorkingDir, ifaces)

	if opts.IncludeFields {
		collectFieldDefs(pkgs, opts.WorkingDir, states, fkeys)
	}

	collectUses(pkgs, states, opts.TreatTestsAsExternal, fkeys)

	results := make([]report.Candidate, 0, len(states))
	for _, key := range slices.Sorted(maps.Keys(states)) {
		state := states[key]
		if len(state.externalRefPkg) > 0 || state.candidate.InternalRefCount == 0 {
			continue
		}
		results = append(results, state.candidate)
	}

	return filterCandidates(results, opts.ExcludePackages, opts.ExcludeSymbols), nil
}

func collectDefinitions(
	pkgs []*packages.Package,
	workingDir string,
	ifaces []namedInterface,
) map[string]*candidateState {
	states := map[string]*candidateState{}

	for _, pkg := range pkgs {
		if !isDefinitionPackage(pkg) {
			continue
		}

		files := buildFileInfo(pkg)
		for ident, obj := range pkg.TypesInfo.Defs {
			meta, ok := fileForPos(pkg.Fset, files, ident.Pos())
			if !ok {
				continue
			}
			if key, ifcs, ok := classifyDef(obj, meta, ifaces); ok {
				states[key] = &candidateState{
					candidate:      newCandidate(key, pkg, obj, meta, pkg.Fset, workingDir, ifcs),
					externalRefPkg: map[string]struct{}{},
				}
			}
		}
	}

	return states
}

func classifyDef(
	obj types.Object,
	meta fileInfo,
	ifaces []namedInterface,
) (string, []namedInterface, bool) {
	if isCandidateObject(obj) {
		if meta.isTest && isGoTestEntrypoint(obj) {
			return "", nil, false
		}
		return objectKey(obj), nil, true
	}
	if len(ifaces) > 0 {
		if key, ok := methodCandidateKey(obj); ok {
			return key, ifaces, true
		}
	}
	return "", nil, false
}

func newCandidate(
	key string,
	pkg *packages.Package,
	obj types.Object,
	meta fileInfo,
	fset *token.FileSet,
	workingDir string,
	ifaces []namedInterface,
) report.Candidate {
	kind := objectKind(obj)
	reasons := candidateReasons(pkg, meta)

	if fn, ok := obj.(*types.Func); ok {
		if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
			kind = "method"
			reasons = append(
				reasons,
				methodInterfaceReasons(sig.Recv(), fn.Name(), ifaces)...,
			)
		}
	}

	return buildCandidate(key, kind, positionString(fset, obj.Pos(), workingDir), reasons)
}

func candidateReasons(pkg *packages.Package, meta fileInfo) []string {
	reasons := []string{}
	if pkg.Name == "main" {
		reasons = append(reasons, "package main")
	}
	if packageUnderCmd(pkg.PkgPath) {
		reasons = append(reasons, "package under cmd")
	}
	if meta.generated {
		reasons = append(reasons, "generated file")
	}
	return reasons
}

func collectUses(
	pkgs []*packages.Package,
	states map[string]*candidateState,
	treatTestsAsExternal bool,
	fkeys *fieldKeyMap,
) {
	for _, pkg := range pkgs {
		if !shouldCollectUsesFromPackage(pkg, treatTestsAsExternal) {
			continue
		}

		files := buildFileInfo(pkg)
		for ident, obj := range pkg.TypesInfo.Uses {
			recordUse(pkg, ident, obj, files, states, treatTestsAsExternal)
		}
		if fkeys != nil {
			recordFieldSelections(pkg, files, states, treatTestsAsExternal, fkeys)
		}
	}
}

func recordFieldSelections(
	pkg *packages.Package,
	files map[string]fileInfo,
	states map[string]*candidateState,
	treatTestsAsExternal bool,
	fkeys *fieldKeyMap,
) {
	for selExpr, sel := range pkg.TypesInfo.Selections {
		state, fieldVar := resolveFieldSelection(sel, fkeys, states)
		if state == nil {
			continue
		}
		applyUse(pkg, selExpr.Sel.Pos(), fieldVar, files, state, treatTestsAsExternal)
	}
}

func resolveFieldSelection(
	sel *types.Selection,
	fkeys *fieldKeyMap,
	states map[string]*candidateState,
) (*candidateState, *types.Var) {
	if sel.Kind() != types.FieldVal {
		return nil, nil
	}
	fieldVar, ok := sel.Obj().(*types.Var)
	if !ok || !fieldVar.IsField() {
		return nil, nil
	}
	fm, ok := fkeys.resolve(fieldVar)
	if !ok {
		return nil, nil
	}
	state, ok := states[fm.key]
	if !ok {
		return nil, nil
	}
	return state, fieldVar
}

func buildFileInfo(pkg *packages.Package) map[string]fileInfo {
	files := make(map[string]fileInfo, len(pkg.Syntax))
	for idx, file := range pkg.Syntax {
		name := pkg.CompiledGoFiles[idx]
		files[name] = fileInfo{
			name:      name,
			isTest:    strings.HasSuffix(name, "_test.go"),
			generated: ast.IsGenerated(file),
		}
	}
	return files
}

func fileForPos(fset *token.FileSet, files map[string]fileInfo, pos token.Pos) (fileInfo, bool) {
	f := fset.File(pos)
	if f == nil {
		return fileInfo{}, false
	}
	meta, ok := files[f.Name()]
	return meta, ok
}

func isDefinitionPackage(pkg *packages.Package) bool {
	return pkg.ForTest == "" && !isExternalTestPackage(pkg)
}

func isExternalTestPackage(pkg *packages.Package) bool {
	return strings.HasSuffix(pkg.Name, "_test")
}

func shouldCollectUsesFromPackage(pkg *packages.Package, treatTestsAsExternal bool) bool {
	if isDefinitionPackage(pkg) {
		return true
	}

	return treatTestsAsExternal && isExternalTestPackage(pkg)
}

func packageUnderCmd(pkgPath string) bool {
	for part := range strings.SplitSeq(pkgPath, "/") {
		if part == "cmd" {
			return true
		}
	}
	return false
}

func isCandidateObject(obj types.Object) bool {
	if obj == nil || !obj.Exported() || obj.Pkg() == nil {
		return false
	}
	if obj.Parent() != obj.Pkg().Scope() {
		return false
	}

	switch current := obj.(type) {
	case *types.Func:
		sig, ok := current.Type().(*types.Signature)
		return ok && sig.Recv() == nil
	case *types.TypeName, *types.Const, *types.Var:
		return true
	default:
		return false
	}
}

func objectKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "func"
	case *types.TypeName:
		return "type"
	case *types.Const:
		return "const"
	case *types.Var:
		return "var"
	default:
		return "unknown"
	}
}

func objectKey(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil {
		return ""
	}

	if fn, ok := obj.(*types.Func); ok {
		if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
			if recvName, exported := receiverTypeName(sig.Recv()); exported {
				return fn.Pkg().Path() + "." + recvName + "." + fn.Name()
			}
		}
	}

	return obj.Pkg().Path() + "." + obj.Name()
}

func positionString(fset *token.FileSet, pos token.Pos, workingDir string) string {
	position := fset.Position(pos)
	name := position.Filename
	if rel, err := filepath.Rel(workingDir, name); err == nil {
		name = rel
	}
	return filepath.ToSlash(fmt.Sprintf("%s:%d", name, position.Line))
}

func filterCandidates(
	candidates []report.Candidate,
	excludePackages []string,
	excludeSymbols []string,
) []report.Candidate {
	if len(excludePackages) == 0 && len(excludeSymbols) == 0 {
		return candidates
	}

	excludedPackages := make(map[string]struct{}, len(excludePackages))
	for _, pkg := range excludePackages {
		excludedPackages[pkg] = struct{}{}
	}

	excludedSymbols := make(map[string]struct{}, len(excludeSymbols))
	for _, symbol := range excludeSymbols {
		excludedSymbols[symbol] = struct{}{}
	}

	filtered := make([]report.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := excludedSymbols[candidate.Symbol]; ok {
			continue
		}
		if len(excludedPackages) > 0 {
			if _, ok := excludedPackages[candidatePackagePath(candidate.Symbol)]; ok {
				continue
			}
		}
		filtered = append(filtered, candidate)
	}

	return filtered
}

func candidatePackagePath(symbol string) string {
	// Symbol format: "pkg/path.Name" or "pkg/path.Type.Method"
	// Find the first dot that separates the package path from the symbol name.
	// Package paths use "/" separators, so the first "." after the last "/" is
	// the boundary.
	lastSlash := strings.LastIndex(symbol, "/")
	dotIdx := strings.Index(symbol[lastSlash+1:], ".")
	if dotIdx == -1 {
		return ""
	}
	return symbol[:lastSlash+1+dotIdx]
}

func samePackage(pkg *packages.Package, obj types.Object) bool {
	return obj.Pkg() != nil && pkg.PkgPath == obj.Pkg().Path()
}

func isGoTestEntrypoint(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() != nil || sig.Results().Len() != 0 {
		return false
	}

	switch {
	case hasGoTestPrefix(fn.Name(), "Test"):
		return hasSingleTestingParam(sig, "T")
	case hasGoTestPrefix(fn.Name(), "Benchmark"):
		return hasSingleTestingParam(sig, "B")
	case hasGoTestPrefix(fn.Name(), "Fuzz"):
		return hasSingleTestingParam(sig, "F")
	case hasGoTestPrefix(fn.Name(), "Example"):
		return sig.Params().Len() == 0
	default:
		return false
	}
}

func hasGoTestPrefix(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) {
		return true
	}

	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}

func hasSingleTestingParam(sig *types.Signature, want string) bool {
	if sig.Params().Len() != 1 {
		return false
	}

	ptr, ok := sig.Params().At(0).Type().(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	return named.Obj().Pkg().Path() == "testing" && named.Obj().Name() == want
}

func recordUse(
	pkg *packages.Package,
	ident *ast.Ident,
	obj types.Object,
	files map[string]fileInfo,
	states map[string]*candidateState,
	treatTestsAsExternal bool,
) {
	key := objectKey(obj)
	if key == "" {
		return
	}

	state, ok := states[key]
	if !ok {
		return
	}

	applyUse(pkg, ident.Pos(), obj, files, state, treatTestsAsExternal)
}

func applyUse(
	pkg *packages.Package,
	pos token.Pos,
	obj types.Object,
	files map[string]fileInfo,
	state *candidateState,
	treatTestsAsExternal bool,
) {
	meta, ok := fileForPos(pkg.Fset, files, pos)
	if !ok || meta.generated {
		return
	}
	if meta.isTest && (!treatTestsAsExternal || !isExternalTestPackage(pkg)) {
		return
	}
	if samePackage(pkg, obj) {
		state.candidate.InternalRefCount++
	} else {
		state.externalRefPkg[pkg.PkgPath] = struct{}{}
	}
}

func receiverTypeName(recv *types.Var) (string, bool) {
	t := recv.Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return "", false
	}
	return named.Obj().Name(), named.Obj().Exported()
}

type namedInterface struct {
	name  string
	iface *types.Interface
}

func collectInterfaces(pkgs []*packages.Package) []namedInterface {
	seen := map[*types.Package]bool{}
	var result []namedInterface

	var visit func(tp *types.Package)
	visit = func(tp *types.Package) {
		if tp == nil || seen[tp] {
			return
		}
		seen[tp] = true

		result = append(result, interfacesInScope(tp)...)
		for _, imp := range tp.Imports() {
			visit(imp)
		}
	}

	for _, pkg := range pkgs {
		if pkg.Types != nil {
			visit(pkg.Types)
		}
	}

	return result
}

func interfacesInScope(tp *types.Package) []namedInterface {
	var result []namedInterface
	scope := tp.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		iface, ok := tn.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		result = append(result, namedInterface{
			name:  tp.Path() + "." + tn.Name(),
			iface: iface,
		})
	}
	return result
}

func methodCandidateKey(obj types.Object) (string, bool) {
	fn, ok := obj.(*types.Func)
	if !ok || !fn.Exported() || fn.Pkg() == nil {
		return "", false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return "", false
	}
	recvName, exported := receiverTypeName(sig.Recv())
	if !exported {
		return "", false
	}
	return fn.Pkg().Path() + "." + recvName + "." + fn.Name(), true
}

type fieldMeta struct {
	key      string
	embedded bool
	tag      string
}

type fieldKeyMap struct {
	byPtr  map[*types.Var]fieldMeta
	byName map[string][]fieldMeta // "pkgPath\x00fieldName" → metas
}

func (fk *fieldKeyMap) resolve(v *types.Var) (fieldMeta, bool) {
	if fk == nil {
		return fieldMeta{}, false
	}
	if fm, ok := fk.byPtr[v]; ok {
		return fm, true
	}
	if v.Pkg() == nil {
		return fieldMeta{}, false
	}
	metas := fk.byName[v.Pkg().Path()+"\x00"+v.Name()]
	if len(metas) == 1 {
		return metas[0], true
	}
	return fieldMeta{}, false
}

func buildFieldKeyMap(pkgs []*packages.Package) *fieldKeyMap {
	fk := &fieldKeyMap{
		byPtr:  map[*types.Var]fieldMeta{},
		byName: map[string][]fieldMeta{},
	}
	for _, pkg := range pkgs {
		if !isDefinitionPackage(pkg) {
			continue
		}
		collectStructFields(pkg.Types.Scope(), fk)
	}
	return fk
}

func collectStructFields(scope *types.Scope, fk *fieldKeyMap) {
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || !tn.Exported() {
			continue
		}
		st, ok := tn.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for i := range st.NumFields() {
			field := st.Field(i)
			if !field.Exported() {
				continue
			}
			key := tn.Pkg().Path() + "." + tn.Name() + "." + field.Name()
			fm := fieldMeta{
				key:      key,
				embedded: field.Embedded(),
				tag:      st.Tag(i),
			}
			fk.byPtr[field] = fm
			nameKey := tn.Pkg().Path() + "\x00" + field.Name()
			fk.byName[nameKey] = append(fk.byName[nameKey], fm)
		}
	}
}

func collectFieldDefs(
	pkgs []*packages.Package,
	workingDir string,
	states map[string]*candidateState,
	fkeys *fieldKeyMap,
) {
	type defPkg struct {
		pkg   *packages.Package
		files map[string]fileInfo
	}
	defPkgs := map[string]defPkg{}
	for _, pkg := range pkgs {
		if isDefinitionPackage(pkg) {
			defPkgs[pkg.PkgPath] = defPkg{pkg: pkg, files: buildFileInfo(pkg)}
		}
	}

	for field, fm := range fkeys.byPtr {
		if _, exists := states[fm.key]; exists {
			continue
		}
		if field.Pkg() == nil {
			continue
		}
		dp, ok := defPkgs[field.Pkg().Path()]
		if !ok {
			continue
		}
		meta, ok := fileForPos(dp.pkg.Fset, dp.files, field.Pos())
		if !ok {
			continue
		}
		states[fm.key] = &candidateState{
			candidate:      newFieldCandidate(fm, dp.pkg, field, meta, dp.pkg.Fset, workingDir),
			externalRefPkg: map[string]struct{}{},
		}
	}
}

func newFieldCandidate(
	fm fieldMeta,
	pkg *packages.Package,
	field *types.Var,
	meta fileInfo,
	fset *token.FileSet,
	workingDir string,
) report.Candidate {
	reasons := candidateReasons(pkg, meta)
	reasons = append(reasons, fieldReasons(fm)...)

	return buildCandidate(fm.key, "field", positionString(fset, field.Pos(), workingDir), reasons)
}

func buildCandidate(symbol, kind, definedIn string, reasons []string) report.Candidate {
	confidence := report.ConfidenceHigh
	if len(reasons) > 0 {
		confidence = report.ConfidenceLow
	}

	return report.Candidate{
		Symbol:              symbol,
		Kind:                kind,
		DefinedIn:           definedIn,
		InternalRefCount:    0,
		ExternalRefPkgCount: 0,
		ExternalRefExamples: []string{},
		Confidence:          confidence,
		Reasons:             reasons,
	}
}

func fieldReasons(fm fieldMeta) []string {
	var reasons []string
	if fm.embedded {
		reasons = append(reasons, "embedded field")
	}
	if hasSerializationTag(fm.tag) {
		reasons = append(reasons, "has serialization tag")
	}
	return reasons
}

func hasSerializationTag(tag string) bool {
	st := reflect.StructTag(tag)
	for _, key := range []string{"json", "xml", "yaml"} {
		if _, ok := st.Lookup(key); ok {
			return true
		}
	}
	return false
}

func methodInterfaceReasons(
	recv *types.Var,
	methodName string,
	ifaces []namedInterface,
) []string {
	t := recv.Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}

	var reasons []string
	ptrT := types.NewPointer(named)

	for _, ni := range ifaces {
		if !types.Implements(named, ni.iface) &&
			!types.Implements(ptrT, ni.iface) {
			continue
		}
		for m := range ni.iface.Methods() {
			if m.Name() == methodName {
				reasons = append(reasons, "satisfies interface "+ni.name)
				break
			}
		}
	}

	return reasons
}
