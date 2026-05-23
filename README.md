# Simulador Keylogger em Go

Lab educacional e controlado para demonstrar o ciclo basico de um keylogger sem capturar teclas globais do sistema.

## Limites de seguranca

- Captura somente caracteres digitados no proprio terminal do programa.
- Nao instala hooks globais de teclado.
- Nao roda oculto, nao persiste no sistema e nao inicia automaticamente.
- Envio por e-mail fica desativado por padrao e precisa de `--email` mais variaveis `SIM_SMTP_*`.
- Use apenas em ambiente autorizado e com dados ficticios.

## O que faltava no lab original

- Credenciais SMTP estavam hardcoded no codigo Python.
- O README estava com encoding quebrado.
- O envio automatico iniciava sempre, mesmo sem configuracao real.
- Faltava uma versao compilavel/portavel sem dependencia do Python.
- O arquivo de evidencias parecia conter dados sensiveis de teste; prefira exemplos sanitizados.

## Arquivos principais

| Arquivo | Funcao |
| --- | --- |
| `main.go` | Loop principal, log, comandos e envio SMTP opt-in |
| `console_reader.go` | Leitura raw do terminal local |
| `simulated_keylogger_console.py` | Versao Python original mantida como referencia |
| `capturas_simuladas.txt` | Log gerado em runtime |

## Como executar

```powershell
go run .
```

Comandos durante a execucao:

| Tecla | Funcao |
| --- | --- |
| `ESC` | Encerra o programa |
| `Ctrl+L` | Limpa o log |
| `Ctrl+S` | Envia o log agora se `--email` estiver ativo |
| Demais teclas | Sao registradas no arquivo de log |

## Envio SMTP de teste

O envio e intencionalmente opt-in:

```powershell
$env:SIM_SMTP_HOST="smtp.example.com"
$env:SIM_SMTP_PORT="587"
$env:SIM_SMTP_USER="conta-de-teste@example.com"
$env:SIM_SMTP_PASS="senha-de-teste"
$env:SIM_SMTP_TO="destino-de-teste@example.com"
go run . --email --interval 1h
```

Tambem da para mudar o arquivo de log:

```powershell
go run . --log capturas_simuladas.txt
```

## Build

```powershell
go build -o lab_keylogger_simulado.exe .
.\lab_keylogger_simulado.exe
```

## Exemplo de log

```text
[2026-05-22T23:00:00Z] key=a
[2026-05-22T23:00:01Z] key=b
[2026-05-22T23:00:02Z] key=<CTRL-13>
```

## Defesa e detecção

Este lab ajuda a discutir controles defensivos contra keyloggers reais:

- Monitoramento de execucao de binarios desconhecidos.
- EDR/antivirus com deteccao comportamental.
- Controle de saida SMTP/HTTP/DNS incomum.
- MFA para reduzir impacto de roubo de senha.
- Treinamento para evitar execucao de anexos e scripts nao confiaveis.
