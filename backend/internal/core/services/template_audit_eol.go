package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	eolCycleRegex = regexp.MustCompile(`^v?(\d+\.\d+|\d+)`)
	eolHTTPClient = &http.Client{Timeout: 5 * time.Second}
	eolCacheMu    sync.RWMutex
	eolCache      = make(map[string]EOLResult)
)

func (w *TemplateAuditWorker) recheckAllEOL(ctx context.Context) {
	templates, err := w.govRepo.ListTemplates(ctx, "", "", "")
	if err != nil {
		log.Printf("[TemplateAuditWorker] Error listando plantillas para chequeo EOL: %v", err)
		return
	}
	for _, item := range templates {
		status, date, msg := checkEOL(item.DockerImage)
		if err := w.govRepo.UpdateEOLStatus(ctx, item.ID, status, date, msg); err != nil {
			log.Printf("[TemplateAuditWorker] Error actualizando EOL de plantilla %s: %v", item.ID, err)
		}
	}
}

func parseProductAndCycle(imageRef string) (product string, cycle string, ok bool) {
	parts := strings.Split(imageRef, ":")
	if len(parts) != 2 {
		return "", "", false
	}
	repo := parts[0]
	tag := parts[1]

	if slashIdx := strings.LastIndex(repo, "/"); slashIdx != -1 {
		repo = repo[slashIdx+1:]
	}

	repo = strings.ToLower(repo)
	switch repo {
	case "python":
		product = "python"
	case "node", "nodejs":
		product = "nodejs"
	case "ubuntu":
		product = "ubuntu"
	case "debian":
		product = "debian"
	case "golang", "go":
		product = "go"
	case "postgres", "postgresql":
		product = "postgresql"
	case "mysql":
		product = "mysql"
	case "redis":
		product = "redis"
	case "alpine":
		product = "alpine"
	default:
		return "", "", false
	}

	match := eolCycleRegex.FindStringSubmatch(tag)
	if len(match) < 2 {
		return "", "", false
	}
	cycle = match[1]
	return product, cycle, true
}

func checkEOL(imageRef string) (string, string, string) {
	eolCacheMu.RLock()
	cached, found := eolCache[imageRef]
	eolCacheMu.RUnlock()
	if found {
		return cached.Status, cached.Date, cached.Message
	}

	product, cycle, ok := parseProductAndCycle(imageRef)
	if !ok {
		res := EOLResult{
			Status:  "supported",
			Date:    "",
			Message: "Ciclo de vida no indexado o gestionado por la comunidad",
		}
		eolCacheMu.Lock()
		eolCache[imageRef] = res
		eolCacheMu.Unlock()
		return res.Status, res.Date, res.Message
	}

	url := fmt.Sprintf("https://endoflife.date/api/%s/%s.json", product, cycle)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		res := EOLResult{
			Status:  "supported",
			Date:    "",
			Message: "Ciclo de vida activo o no indexado",
		}
		return res.Status, res.Date, res.Message
	}
	resp, err := eolHTTPClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			_ = resp.Body.Close()
		}
		res := EOLResult{
			Status:  "supported",
			Date:    "",
			Message: "Ciclo de vida activo o no indexado",
		}
		return res.Status, res.Date, res.Message
	}
	defer func() { _ = resp.Body.Close() }()

	var payload struct {
		Cycle string          `json:"cycle"`
		EOL   json.RawMessage `json:"eol"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		res := EOLResult{Status: "supported", Date: "", Message: "Soporte activo"}
		return res.Status, res.Date, res.Message
	}

	rawEOL := strings.Trim(string(payload.EOL), `"`)
	now := time.Now()
	var res EOLResult

	switch rawEOL {
	case "true":
		res = EOLResult{
			Status:  "eol",
			Date:    "",
			Message: fmt.Sprintf("Versión %s %s sin soporte oficial (fin de ciclo de vida)", product, cycle),
		}
	case "false", "":
		res = EOLResult{
			Status:  "supported",
			Date:    "",
			Message: fmt.Sprintf("Versión %s %s con soporte oficial activo", product, cycle),
		}
	default:
		parsedDate, parseErr := time.Parse("2006-01-02", rawEOL)
		if parseErr != nil {
			res = EOLResult{
				Status:  "supported",
				Date:    rawEOL,
				Message: fmt.Sprintf("Soporte oficial hasta %s", rawEOL),
			}
		} else {
			if parsedDate.Before(now) {
				res = EOLResult{
					Status:  "eol",
					Date:    rawEOL,
					Message: fmt.Sprintf("Versión sin soporte oficial (finalizó el %s)", rawEOL),
				}
			} else if parsedDate.Before(now.AddDate(0, 6, 0)) {
				res = EOLResult{
					Status:  "warning",
					Date:    rawEOL,
					Message: fmt.Sprintf("Próximo a fin de soporte oficial (finaliza el %s)", rawEOL),
				}
			} else {
				res = EOLResult{
					Status:  "supported",
					Date:    rawEOL,
					Message: fmt.Sprintf("Soporte oficial activo hasta %s", rawEOL),
				}
			}
		}
	}

	eolCacheMu.Lock()
	eolCache[imageRef] = res
	eolCacheMu.Unlock()

	return res.Status, res.Date, res.Message
}
