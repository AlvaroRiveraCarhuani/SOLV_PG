package integration

import (
	"os"
	"path/filepath"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestASTSecurityViolationAllLanguages(t *testing.T) {
	testCases := []struct {
		lang     string
		filePath string
		rules    domain.ASTRules
	}{
		{
			lang:     "python",
			filePath: filepath.Join("testdata", "algoritmia", "python", "ast_forbidden.py"),
			rules:    domain.ASTRules{ForbiddenImports: []string{"os", "sys"}, ForbiddenFunctions: []string{"eval"}},
		},
		{
			lang:     "c",
			filePath: filepath.Join("testdata", "algoritmia", "c", "ast_forbidden.c"),
			rules:    domain.ASTRules{ForbiddenImports: []string{"stdlib.h"}, ForbiddenFunctions: []string{"system"}},
		},
		{
			lang:     "cpp",
			filePath: filepath.Join("testdata", "algoritmia", "cpp", "ast_forbidden.cpp"),
			rules:    domain.ASTRules{ForbiddenImports: []string{"fstream"}, ForbiddenFunctions: []string{"system"}},
		},
		{
			lang:     "csharp",
			filePath: filepath.Join("testdata", "algoritmia", "csharp", "ast_forbidden.cs"),
			rules:    domain.ASTRules{ForbiddenImports: []string{"IO"}, ForbiddenFunctions: []string{"ReadAllText"}},
		},
		{
			lang:     "java",
			filePath: filepath.Join("testdata", "algoritmia", "java", "ast_forbidden.java"),
			rules:    domain.ASTRules{ForbiddenImports: []string{"io.File"}, ForbiddenFunctions: []string{"delete"}},
		},
		{
			lang:     "javascript",
			filePath: filepath.Join("testdata", "algoritmia", "javascript", "ast_forbidden.js"),
			rules:    domain.ASTRules{ForbiddenImports: []string{"fs"}, ForbiddenFunctions: []string{"eval"}},
		},
	}

	astAnalyzer := services.NewStaticASTAnalyzer()

	for _, tc := range testCases {
		t.Run(tc.lang, func(t *testing.T) {
			codeBytes, err := os.ReadFile(tc.filePath)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", tc.lang, tc.filePath, err)
			}

			ok, msg := astAnalyzer.ValidateCode(tc.lang, string(codeBytes), tc.rules)
			if ok {
				t.Errorf("[%s] Se esperaba que la validación AST FALLARA para código prohibido, pero PASÓ", tc.lang)
			} else {
				t.Logf("[%s] Violación AST detectada exitosamente: %s", tc.lang, msg)
			}
		})
	}
}
