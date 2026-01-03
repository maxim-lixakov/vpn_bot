package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vpn-app/internal/telegram"
	"vpn-app/internal/utils"
)

type sendLogsResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleSendLogs(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BackupAdminTgUserID == 0 {
		utils.WriteJSON(w, sendLogsResp{
			Success: false,
			Error:   "BACKUP_ADMIN_TG_USER_ID is not set",
		})
		return
	}

	if s.cfg.BotToken == "" {
		utils.WriteJSON(w, sendLogsResp{
			Success: false,
			Error:   "BOT_TOKEN is not set",
		})
		return
	}

	ctx := r.Context()

	// Вычисляем диапазон последних 3 дней
	now := time.Now().UTC()

	// Собираем логи за последние 3 дня
	lastWeekStart := now.AddDate(0, 0, -3)
	lastWeekEnd := now

	// Формируем имя файла: DD.MM.YYYY_HH:MM-DD.MM.YYYY_HH:MM.log
	filename := fmt.Sprintf("%s-%s.log",
		lastWeekStart.Format("02.01.2006_15:04"),
		lastWeekEnd.Format("02.01.2006_15:04"))

	// Создаем временную директорию для логов
	tmpDir, err := os.MkdirTemp("", "logs_*")
	if err != nil {
		http.Error(w, "failed to create temp dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	logFilePath := filepath.Join(tmpDir, filename)

	// Собираем логи из всех контейнеров за последние 3 дня
	// Используем docker ps для получения списка контейнеров проекта
	// Получаем имя проекта из переменной окружения или используем дефолтное
	projectName := os.Getenv("COMPOSE_PROJECT_NAME")
	if projectName == "" {
		projectName = "vpn_bot" // Дефолтное имя проекта
	}

	// Получаем список контейнеров проекта
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.Names}}", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	containersOutput, err := cmd.Output()
	if err != nil {
		log.Printf("failed to get container list: %v, trying alternative method", err)
		// Альтернативный способ: получаем все контейнеры и фильтруем по имени
		cmd = exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.Names}}")
		containersOutput, err = cmd.Output()
		if err != nil {
			log.Printf("failed to get container list (alternative): %v", err)
			// Fallback на известные имена контейнеров
			containersOutput = []byte(fmt.Sprintf("%s-app-1\n%s-telegram-bot-1\n%s-periodic-tasks-1", projectName, projectName, projectName))
		} else {
			// Фильтруем контейнеры по имени проекта
			allContainers := strings.Split(strings.TrimSpace(string(containersOutput)), "\n")
			var filtered []string
			for _, container := range allContainers {
				if strings.Contains(container, projectName) && (strings.Contains(container, "app") || strings.Contains(container, "telegram-bot") || strings.Contains(container, "periodic-tasks")) {
					filtered = append(filtered, container)
				}
			}
			containersOutput = []byte(strings.Join(filtered, "\n"))
		}
	}

	containerNames := strings.Split(strings.TrimSpace(string(containersOutput)), "\n")

	log.Printf("found %d containers: %v", len(containerNames), containerNames)
	log.Printf("collecting logs from %s to %s", lastWeekStart.Format(time.RFC3339), lastWeekEnd.Format(time.RFC3339))

	var allLogs strings.Builder

	// Форматируем время для docker logs (since и until)
	sinceStr := lastWeekStart.Format(time.RFC3339)
	untilStr := lastWeekEnd.Format(time.RFC3339)

	for _, container := range containerNames {
		container = strings.TrimSpace(container)
		if container == "" {
			continue
		}

		// Получаем логи контейнера за период
		cmd := exec.CommandContext(ctx, "docker", "logs",
			"--since", sinceStr,
			"--until", untilStr,
			container)

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Логируем ошибки для отладки
			log.Printf("failed to get logs from container %s: %v, output: %s", container, err, string(output))
			// Пробуем получить логи без фильтра по времени (для отладки)
			cmdNoTime := exec.CommandContext(ctx, "docker", "logs", "--tail", "100", container)
			outputNoTime, errNoTime := cmdNoTime.CombinedOutput()
			if errNoTime == nil && len(outputNoTime) > 0 {
				log.Printf("container %s has logs (last 100 lines), but filtered query failed", container)
				allLogs.WriteString(fmt.Sprintf("=== %s (last 100 lines, time filter failed) ===\n", container))
				allLogs.Write(outputNoTime)
				allLogs.WriteString("\n\n")
			}
			continue
		}

		if len(output) > 0 {
			allLogs.WriteString(fmt.Sprintf("=== %s ===\n", container))
			allLogs.Write(output)
			allLogs.WriteString("\n\n")
		}
	}

	// Если логов нет, возвращаем сообщение
	if allLogs.Len() == 0 {
		utils.WriteJSON(w, sendLogsResp{
			Success: true,
			Message: "No logs found for the past 3 days",
		})
		return
	}

	// Записываем логи в файл
	if err := os.WriteFile(logFilePath, []byte(allLogs.String()), 0644); err != nil {
		http.Error(w, "failed to write log file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Читаем файл для отправки
	logData, err := os.ReadFile(logFilePath)
	if err != nil {
		http.Error(w, "failed to read log file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем caption
	caption := fmt.Sprintf(
		"📋 Logs за последние 3 дня\n\n"+
			"Период: %s - %s\n"+
			"Размер: %.2f MB",
		lastWeekStart.Format("02.01.2006 15:04"),
		lastWeekEnd.Format("02.01.2006 15:04"),
		float64(len(logData))/(1024*1024),
	)

	// Отправляем логи админу
	log.Printf("sending logs to Telegram user ID: %d", s.cfg.BackupAdminTgUserID)
	if err := telegram.SendDocument(
		s.cfg.BotToken,
		s.cfg.BackupAdminTgUserID,
		filename,
		logData,
		caption,
	); err != nil {
		log.Printf("failed to send logs to Telegram: %v", err)
		utils.WriteJSON(w, sendLogsResp{
			Success: false,
			Error:   fmt.Sprintf("failed to send telegram document: %v", err),
		})
		return
	}

	// Docker автоматически управляет логами через настройки max-size и max-file в docker-compose.yml
	// Старые логи будут автоматически удаляться при достижении лимитов
	log.Printf("logs sent successfully")

	utils.WriteJSON(w, sendLogsResp{
		Success: true,
		Message: fmt.Sprintf("Logs sent successfully. Period: %s - %s, Size: %.2f MB",
			lastWeekStart.Format("02.01.2006 15:04"),
			lastWeekEnd.Format("02.01.2006 15:04"),
			float64(len(logData))/(1024*1024)),
	})
}
