# Alexa Attendance Skill

Controle de presenca por voz com Amazon Alexa. Basta falar o nome da pessoa para registrar chegada ou saida, com tudo salvo automaticamente no Google Sheets.

## Exemplo de Uso

> "Alexa, abrir controle de presenca"
> "Joao Silva chegou"
> "Entendido. Joao Silva foi registrado como presente."

## Visao Geral

A skill roda em AWS Lambda e grava os eventos de presenca em uma planilha do Google Sheets. Cada dia recebe sua propria aba, com colunas para **Nome**, **Chegada** e **Saida**.

Uma segunda funcao Lambda roda por agendamento para criar automaticamente a aba do proximo dia antes do inicio do expediente.

### Intencoes Suportadas

| Acao | Exemplos |
|---|---|
| Registrar chegada | "Joao chegou", "registrar chegada de Joao", "marcar Joao como presente" |
| Registrar saida | "Joao saiu", "registrar saida de Joao", "Joao esta saindo" |

Quando uma saida e registrada, a skill atualiza a ultima linha aberta de chegada daquela pessoa. Se nenhuma chegada existir, a Alexa pede confirmacao antes de salvar uma saida avulsa.

## Arquitetura

```text
Dispositivo Alexa  ->  Servico Alexa  ->  Lambda (Go, ARM64)  ->  Google Sheets API
                                                |
                                           AWS Secrets Manager
                                           (credenciais Google)

EventBridge (cron diario)  ->  Lambda Sheet-Setup  ->  Google Sheets API
```

**Stack tecnologico:** Go, AWS Lambda, AWS CDK (TypeScript), Google Sheets API v4, Alexa Skills Kit

## Pre-requisitos

- Conta AWS com permissoes para Lambda, IAM, EventBridge, CloudFormation e Secrets Manager
- Conta de desenvolvedor Amazon para a skill Alexa
- Projeto no Google Cloud com a API do Sheets habilitada
- Service account do Google com acesso de Editor na planilha de destino
- Go 1.25+
- Node.js 18+
- GNU Make ou PowerShell
- ASK CLI para sincronizar o modelo de interacao pela linha de comando

## Fluxo Recomendado de Setup

### 1. Preparar o Google Sheets

1. Crie a planilha de destino.
2. Crie uma service account no Google Cloud com a API do Sheets habilitada.
3. Compartilhe a planilha com o email da service account como **Editor**.
4. Baixe o arquivo JSON de credenciais como `credentials.json`.

### 2. Armazenar o `credentials.json` no AWS Secrets Manager

Crie o secret na primeira vez:

```bash
aws secretsmanager create-secret \
  --name alexa-skill/google-credentials \
  --secret-string file://credentials.json \
  --region sa-east-1
```

Atualize o secret depois, se ele ja existir:

```bash
aws secretsmanager put-secret-value \
  --secret-id alexa-skill/google-credentials \
  --secret-string file://credentials.json \
  --region sa-east-1
```

Se preferir, `make store-secret` cobre o fluxo de criacao inicial.

### 3. Criar e Configurar a Skill Alexa

1. Crie uma skill custom no [Console de Desenvolvedor Alexa](https://developer.amazon.com/alexa/console/ask).
2. Defina o nome de invocacao como `controle de presenca` ou a variacao que preferir.
3. Use o modelo de interacao em `skill-package/interactionModels/pt-BR.json`.
4. Depois do deploy, configure o endpoint da skill com o ARN da Lambda.

### 4. Compilar os Artefatos Lambda

```bash
make build-all-artifacts
```

Execute em Linux ou WSL para gerar os binarios em `linux/arm64`.

### 5. Fazer Deploy com CDK

O fluxo com CDK e o caminho recomendado.

Se este for um deploy completamente novo em uma conta/regiao AWS, faca o bootstrap do CDK antes:

```bash
cd infra/cdk
npx cdk bootstrap aws://ACCOUNT_ID/sa-east-1
```

Depois, faca o deploy:

```bash
make deploy-cdk \
  GOOGLE_SHEET_ID=id-da-sua-planilha \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx
```

> Importante: sempre que voce executar comandos diretos do CDK, como `cdk synth`, `cdk diff` ou `cdk deploy`, nao esqueca de passar `-c googleSheetId=xxx`. A aplicacao CDK depende desse contexto.

Exemplo:

```bash
cd infra/cdk
npx cdk diff \
  -c googleSheetId=id-da-sua-planilha \
  -c alexaSkillId=amzn1.ask.skill.xxx \
  -c googleCredentialsSecretName=alexa-skill/google-credentials \
  -c region=sa-east-1
```

### 6. Sincronizar o Modelo de Interacao com a ASK CLI

Depois de criar a skill, ou sempre que `skill-package/interactionModels/pt-BR.json` mudar, envie o modelo atualizado com a ASK CLI:

```bash
ask smapi set-interaction-model -s amzn1.ask.skill.id -g development -l pt-BR --interaction-model "$(cat skill-package/interactionModels/pt-BR.json)"
```

Substitua `amzn1.ask.skill.id` pelo ID real da sua skill.

## Outras Opcoes de Deploy

### CloudFormation

```bash
make deploy-cfn \
  GOOGLE_SHEET_ID=id-da-sua-planilha \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx \
  S3_BUCKET=seu-bucket-de-deploy
```

### Lambda CLI direta

```bash
# Criacao inicial
make create-lambda \
  GOOGLE_SHEET_ID=id-da-sua-planilha \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx

# Atualizacoes posteriores
make deploy-lambda
```

### PowerShell

```powershell
.\deploy.ps1 deploy-cdk -GoogleSheetId "id-da-sua-planilha" -AlexaSkillId "amzn1.ask.skill.xxx"
```

## Configuracao

### Variaveis de Ambiente

| Variavel | Descricao | Padrao |
|---|---|---|
| `GOOGLE_SHEET_ID` | ID da planilha do Google Sheets | obrigatorio |
| `ALEXA_SKILL_ID` | ID da skill Alexa usado na validacao de requisicoes | opcional |
| `GOOGLE_CREDENTIALS_SECRET` | Nome do secret no Secrets Manager | `alexa-skill/google-credentials` |
| `TZ` | Fuso horario usado nos registros | `America/Sao_Paulo` |

### Variaveis Comuns do Make

| Variavel | Descricao | Padrao |
|---|---|---|
| `REGION` | Regiao AWS | `sa-east-1` |
| `FUNCTION_NAME` | Nome da Lambda principal | `alexa-attendance-skill` |
| `SECRET_NAME` | Nome do secret no Secrets Manager | `alexa-skill/google-credentials` |

## Estrutura do Projeto

```text
alexa-attendance-skill/
|-- cmd/
|   |-- lambda/          # Entry point da Lambda da skill Alexa
|   `-- sheet-setup/     # Lambda diaria para criacao de aba
|-- internal/
|   |-- alexa/           # Tratamento de requests e responses da Alexa
|   |-- awsutil/         # Integracao com AWS Secrets Manager
|   `-- sheets/          # Logica cliente do Google Sheets
|-- infra/
|   |-- cdk/             # Stack AWS CDK
|   `-- cloudformation/  # Template CloudFormation
|-- skill-package/       # Manifesto da skill e modelo de interacao
|-- Makefile             # Comandos de build e deploy
`-- deploy.ps1           # Helper de deploy em PowerShell
```

## Testes

```bash
go test ./...
```

A suite de testes cobre o tratamento de requisicoes Alexa, fluxo de sessao, validacao e operacoes no Google Sheets com uso de mocks.

## Licenca

Este projeto e disponibilizado como esta para uso pessoal e educacional.
