package services

import (
	"fmt"
	"regexp"
	"strings"

	"solv-backend/internal/core/domain"
)

type StaticASTAnalyzer struct{}

func NewStaticASTAnalyzer() domain.ASTAnalyzer {
	return &StaticASTAnalyzer{}
}

func (a *StaticASTAnalyzer) ValidateCode(language string, sourceCode string, rules domain.ASTRules) (bool, string) {
	lang := strings.ToLower(strings.TrimSpace(language))

	switch lang {
	case "python", "py":
		return a.validatePython(sourceCode, rules)
	case "cpp", "c++", "c":
		return a.validateCpp(sourceCode, rules)
	case "csharp", "c#", "cs":
		return a.validateCSharp(sourceCode, rules)
	case "java":
		return a.validateJava(sourceCode, rules)
	case "javascript", "js", "node":
		return a.validateJS(sourceCode, rules)
	default:
		return a.validateGeneric(sourceCode, rules)
	}
}

func (a *StaticASTAnalyzer) validatePython(code string, rules domain.ASTRules) (bool, string) {
	for _, imp := range rules.ForbiddenImports {
		impTrim := strings.TrimSpace(imp)
		if impTrim == "" {
			continue
		}

		reImport := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(import\s+(.*\b)?%s\b|from\s+%s\b\s+import)`, regexp.QuoteMeta(impTrim), regexp.QuoteMeta(impTrim)))
		reDynamicImport := regexp.MustCompile(fmt.Sprintf(`__import__\s*\(\s*['"]%s['"]\s*\)`, regexp.QuoteMeta(impTrim)))

		if reImport.MatchString(code) || reDynamicImport.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (Python): Importación prohibida de '%s'", impTrim)
		}
	}

	for _, fn := range rules.ForbiddenFunctions {
		fnTrim := strings.TrimSpace(fn)
		if fnTrim == "" {
			continue
		}

		reFunc := regexp.MustCompile(fmt.Sprintf(`\b%s\s*\(`, regexp.QuoteMeta(fnTrim)))
		if reFunc.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (Python): Uso de función prohibida '%s()'", fnTrim)
		}
	}

	return true, ""
}

func (a *StaticASTAnalyzer) validateCpp(code string, rules domain.ASTRules) (bool, string) {
	for _, imp := range rules.ForbiddenImports {
		impTrim := strings.TrimSpace(imp)
		if impTrim == "" {
			continue
		}

		reInclude := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*#\s*include\s*[<"]%s[>"]`, regexp.QuoteMeta(impTrim)))
		if reInclude.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (C/C++): Cabecera prohibida '#include <%s>'", impTrim)
		}
	}

	for _, fn := range rules.ForbiddenFunctions {
		fnTrim := strings.TrimSpace(fn)
		if fnTrim == "" {
			continue
		}

		reFunc := regexp.MustCompile(fmt.Sprintf(`\b%s\s*\(`, regexp.QuoteMeta(fnTrim)))
		if reFunc.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (C/C++): Uso de función prohibida '%s()'", fnTrim)
		}
	}

	return true, ""
}

func (a *StaticASTAnalyzer) validateCSharp(code string, rules domain.ASTRules) (bool, string) {
	for _, imp := range rules.ForbiddenImports {
		impTrim := strings.TrimSpace(imp)
		if impTrim == "" {
			continue
		}

		reUsing := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*using\s+(System\.)?%s(\b|;|\.)`, regexp.QuoteMeta(impTrim)))
		if reUsing.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (C#): Namespace prohibido 'using %s'", impTrim)
		}
	}

	for _, fn := range rules.ForbiddenFunctions {
		fnTrim := strings.TrimSpace(fn)
		if fnTrim == "" {
			continue
		}

		reFunc := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(fnTrim)))
		if reFunc.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (C#): Uso de método prohibido '%s'", fnTrim)
		}
	}

	return true, ""
}

func (a *StaticASTAnalyzer) validateJava(code string, rules domain.ASTRules) (bool, string) {
	for _, imp := range rules.ForbiddenImports {
		impTrim := strings.TrimSpace(imp)
		if impTrim == "" {
			continue
		}

		reImport := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*import\s+(java\.)?%s(\b|;|\.\*)`, regexp.QuoteMeta(impTrim)))
		if reImport.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (Java): Importación prohibida de '%s'", impTrim)
		}
	}

	for _, fn := range rules.ForbiddenFunctions {
		fnTrim := strings.TrimSpace(fn)
		if fnTrim == "" {
			continue
		}

		reFunc := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(fnTrim)))
		if reFunc.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (Java): Uso de método prohibido '%s'", fnTrim)
		}
	}

	return true, ""
}

func (a *StaticASTAnalyzer) validateJS(code string, rules domain.ASTRules) (bool, string) {
	for _, imp := range rules.ForbiddenImports {
		impTrim := strings.TrimSpace(imp)
		if impTrim == "" {
			continue
		}

		reReq := regexp.MustCompile(fmt.Sprintf(`require\s*\(\s*['"]%s['"]\s*\)|import\s+.*\s+from\s+['"]%s['"]`, regexp.QuoteMeta(impTrim), regexp.QuoteMeta(impTrim)))
		if reReq.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (JS): Módulo prohibido '%s'", impTrim)
		}
	}

	for _, fn := range rules.ForbiddenFunctions {
		fnTrim := strings.TrimSpace(fn)
		if fnTrim == "" {
			continue
		}

		reFunc := regexp.MustCompile(fmt.Sprintf(`\b%s\s*\(`, regexp.QuoteMeta(fnTrim)))
		if reFunc.MatchString(code) {
			return false, fmt.Sprintf("Violación de seguridad AST (JS): Uso de función prohibida '%s()'", fnTrim)
		}
	}

	return true, ""
}

func (a *StaticASTAnalyzer) validateGeneric(code string, rules domain.ASTRules) (bool, string) {
	for _, imp := range rules.ForbiddenImports {
		impTrim := strings.TrimSpace(imp)
		if impTrim != "" && strings.Contains(code, impTrim) {
			return false, fmt.Sprintf("Violación de seguridad AST: Importación prohibida de '%s'", impTrim)
		}
	}

	for _, fn := range rules.ForbiddenFunctions {
		fnTrim := strings.TrimSpace(fn)
		if fnTrim != "" && strings.Contains(code, fnTrim) {
			return false, fmt.Sprintf("Violación de seguridad AST: Uso de función prohibida '%s'", fnTrim)
		}
	}

	return true, ""
}
