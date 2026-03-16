package scan

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
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
}

type candidateState struct {
	candidate           report.Candidate
	externalRefPkg      map[string]struct{}
	externalRefExamples map[string]struct{}
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

	states := collectDefinitions(pkgs, opts.WorkingDir)
	collectUses(pkgs, states, opts.WorkingDir, opts.TreatTestsAsExternal)

	results := make([]report.Candidate, 0, len(states))
	for _, key := range slices.Sorted(maps.Keys(states)) {
		state := states[key]
		state.candidate.ExternalRefPkgCount = len(state.externalRefPkg)
		state.candidate.ExternalRefExamples = sortedExamples(state.externalRefExamples)
		results = append(results, state.candidate)
	}

	return filterCandidates(results, opts.ExcludePackages, opts.ExcludeSymbols), nil
}

func collectDefinitions(pkgs []*packages.Package, workingDir string) map[string]*candidateState {
	states := map[string]*candidateState{}

	for _, pkg := range pkgs {
		if !isDefinitionPackage(pkg) {
			continue
		}

		files := buildFileInfo(pkg)
		for ident, obj := range pkg.TypesInfo.Defs {
			meta, ok := fileForPos(pkg.Fset, files, ident.Pos())
			if !ok || !isCandidateObject(obj) {
				continue
			}

			key := objectKey(obj)
			states[key] = &candidateState{
				candidate:           newCandidate(key, pkg, obj, meta, pkg.Fset, workingDir),
				externalRefPkg:      map[string]struct{}{},
				externalRefExamples: map[string]struct{}{},
			}
		}
	}

	return states
}

func newCandidate(
	key string,
	pkg *packages.Package,
	obj types.Object,
	meta fileInfo,
	fset *token.FileSet,
	workingDir string,
) report.Candidate {
	reasons := candidateReasons(pkg, obj, meta)
	confidence := report.ConfidenceHigh
	if len(reasons) > 0 {
		confidence = report.ConfidenceLow
	}

	return report.Candidate{
		Symbol:              key,
		Kind:                objectKind(obj),
		DefinedIn:           positionString(fset, obj.Pos(), workingDir),
		InternalRefCount:    0,
		ExternalRefPkgCount: 0,
		ExternalRefExamples: []string{},
		Confidence:          confidence,
		Reasons:             reasons,
	}
}

func candidateReasons(pkg *packages.Package, obj types.Object, meta fileInfo) []string {
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
	if meta.isTest && isGoTestEntrypoint(obj) {
		reasons = append(reasons, "go test entrypoint")
	}

	return reasons
}

func collectUses(
	pkgs []*packages.Package,
	states map[string]*candidateState,
	workingDir string,
	treatTestsAsExternal bool,
) {
	for _, pkg := range pkgs {
		if !shouldCollectUsesFromPackage(pkg, treatTestsAsExternal) {
			continue
		}

		files := buildFileInfo(pkg)
		for ident, obj := range pkg.TypesInfo.Uses {
			recordUse(pkg, ident, obj, files, states, workingDir, treatTestsAsExternal)
		}
	}
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
	idx := strings.LastIndex(symbol, ".")
	if idx == -1 {
		return ""
	}

	return symbol[:idx]
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
	workingDir string,
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

	meta, ok := fileForPos(pkg.Fset, files, ident.Pos())
	if !ok || meta.generated {
		return
	}
	if meta.isTest && (!treatTestsAsExternal || !isExternalTestPackage(pkg)) {
		return
	}

	if samePackage(pkg, obj) {
		state.candidate.InternalRefCount++
		return
	}

	state.externalRefPkg[pkg.PkgPath] = struct{}{}
	state.externalRefExamples[positionString(pkg.Fset, ident.Pos(), workingDir)] = struct{}{}
}

func sortedExamples(examples map[string]struct{}) []string {
	if len(examples) == 0 {
		return []string{}
	}

	return slices.Sorted(maps.Keys(examples))
}
