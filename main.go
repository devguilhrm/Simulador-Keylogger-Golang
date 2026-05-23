package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultLogFile = "capturas_simuladas.txt"

type appConfig struct {
	logFile       string
	enableEmail   bool
	emailInterval time.Duration
	smtp          smtpConfig
}

type smtpConfig struct {
	host     string
	port     int
	username string
	password string
	to       string
}

func main() {
	cfg := loadConfig()

	reader, err := newConsoleReader()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao preparar o terminal: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	if cfg.enableEmail {
		if err := cfg.smtp.validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Config SMTP incompleta: %v\n", err)
			os.Exit(1)
		}
		startEmailScheduler(cfg)
	} else {
		fmt.Println("Email desativado por padrao. Use --email com variaveis SIM_SMTP_* para testar envio.")
	}

	printInstructions(cfg)

	for {
		key, err := reader.ReadKey()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Fprintf(os.Stderr, "\nFalha ao ler tecla: %v\n", err)
			break
		}

		switch key.Code {
		case 27:
			fmt.Println("\nESC detectado. Encerrando.")
			return
		case 12:
			if err := clearLog(cfg.logFile); err != nil {
				fmt.Printf("\nFalha ao limpar log: %v\n", err)
				continue
			}
			fmt.Print("\n[LOG LIMPO]\n")
			continue
		case 19:
			fmt.Print("\n[ENVIO MANUAL SOLICITADO]\n")
			if !cfg.enableEmail {
				fmt.Print("Email esta desativado. Rode com --email e SIM_SMTP_* para testar.\n")
				continue
			}
			go func() {
				if err := sendLogByEmail(cfg); err != nil {
					fmt.Printf("Falha ao enviar email: %v\n", err)
				}
			}()
			continue
		}

		fmt.Print(key.Display)
		if err := appendToLog(cfg.logFile, fmt.Sprintf("key=%s", key.Display)); err != nil {
			fmt.Fprintf(os.Stderr, "\nFalha ao gravar log: %v\n", err)
		}
	}

	fmt.Println("Programa finalizado. Saida limpa.")
}

func loadConfig() appConfig {
	logFile := flag.String("log", defaultLogFile, "arquivo de log gerado pela simulacao")
	enableEmail := flag.Bool("email", false, "habilita envio SMTP de teste; requer SIM_SMTP_HOST, SIM_SMTP_USER, SIM_SMTP_PASS e SIM_SMTP_TO")
	interval := flag.Duration("interval", time.Hour, "intervalo de envio automatico quando --email esta ativo")
	flag.Parse()

	port := 587
	if raw := os.Getenv("SIM_SMTP_PORT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			port = parsed
		}
	}

	return appConfig{
		logFile:       *logFile,
		enableEmail:   *enableEmail,
		emailInterval: *interval,
		smtp: smtpConfig{
			host:     os.Getenv("SIM_SMTP_HOST"),
			port:     port,
			username: os.Getenv("SIM_SMTP_USER"),
			password: os.Getenv("SIM_SMTP_PASS"),
			to:       os.Getenv("SIM_SMTP_TO"),
		},
	}
}

func (cfg smtpConfig) validate() error {
	missing := make([]string, 0)
	if cfg.host == "" {
		missing = append(missing, "SIM_SMTP_HOST")
	}
	if cfg.username == "" {
		missing = append(missing, "SIM_SMTP_USER")
	}
	if cfg.password == "" {
		missing = append(missing, "SIM_SMTP_PASS")
	}
	if cfg.to == "" {
		missing = append(missing, "SIM_SMTP_TO")
	}
	if len(missing) > 0 {
		return fmt.Errorf("faltando %s", strings.Join(missing, ", "))
	}
	return nil
}

func printInstructions(cfg appConfig) {
	fmt.Println("Simulador de captura seguro: registra somente teclas digitadas neste terminal.")
	fmt.Printf("Log: %s\n", cfg.logFile)
	fmt.Println("Comandos:")
	fmt.Println("  ESC      encerra o programa")
	fmt.Println("  Ctrl+L   limpa o arquivo de log")
	fmt.Println("  Ctrl+S   tenta enviar o log agora, somente se --email estiver ativo")
	fmt.Println("Iniciando captura local...")
}

func appendToLog(logFile, text string) error {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = fmt.Fprintf(file, "[%s] %s\n", timestamp, text)
	return err
}

func clearLog(logFile string) error {
	return os.WriteFile(logFile, []byte{}, 0o600)
}

func startEmailScheduler(cfg appConfig) {
	fmt.Printf("Scheduler iniciado: envio de teste a cada %s.\n", cfg.emailInterval)
	go func() {
		ticker := time.NewTicker(cfg.emailInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := sendLogByEmail(cfg); err != nil {
				fmt.Printf("Falha no envio agendado: %v\n", err)
			}
		}
	}()
}

func sendLogByEmail(cfg appConfig) error {
	data, err := os.ReadFile(cfg.logFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("nenhum arquivo de log encontrado")
		}
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("arquivo de log vazio")
	}

	message, contentType, err := buildEmailMessage(cfg, data)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", cfg.smtp.host, cfg.smtp.port)
	auth := smtp.PlainAuth("", cfg.smtp.username, cfg.smtp.password, cfg.smtp.host)

	conn, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if ok, _ := conn.Extension("STARTTLS"); ok {
		if err := conn.StartTLS(&tls.Config{ServerName: cfg.smtp.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if err := conn.Auth(auth); err != nil {
		return err
	}
	if err := conn.Mail(cfg.smtp.username); err != nil {
		return err
	}
	if err := conn.Rcpt(cfg.smtp.to); err != nil {
		return err
	}

	writer, err := conn.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(strings.ReplaceAll(contentType, "\n", "\r\n"))); err != nil {
		writer.Close()
		return err
	}
	if _, err := writer.Write(message.Bytes()); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return conn.Quit()
}

func buildEmailMessage(cfg appConfig, logData []byte) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(body)

	textPart, err := multipartWriter.CreatePart(map[string][]string{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := io.WriteString(textPart, "Arquivo anexo com capturas simuladas em ambiente de teste.\n"); err != nil {
		return nil, "", err
	}

	attachmentHeader := map[string][]string{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, cfg.logFile)},
		"Content-Transfer-Encoding": {"base64"},
	}
	attachmentPart, err := multipartWriter.CreatePart(attachmentHeader)
	if err != nil {
		return nil, "", err
	}

	encoder := base64.NewEncoder(base64.StdEncoding, attachmentPart)
	if _, err := encoder.Write(logData); err != nil {
		encoder.Close()
		return nil, "", err
	}
	if err := encoder.Close(); err != nil {
		return nil, "", err
	}
	if err := multipartWriter.Close(); err != nil {
		return nil, "", err
	}

	headers := &strings.Builder{}
	fmt.Fprintf(headers, "From: %s\n", cfg.smtp.username)
	fmt.Fprintf(headers, "To: %s\n", cfg.smtp.to)
	fmt.Fprintf(headers, "Subject: Relatorio simulador - %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(headers, "MIME-Version: 1.0\n")
	fmt.Fprintf(headers, "Content-Type: multipart/mixed; boundary=%s\n\n", multipartWriter.Boundary())

	return body, headers.String(), nil
}
