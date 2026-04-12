# Alexa Attendance Skill

Um sistema de controle de presenca por voz para Amazon Alexa. Registre chegadas e saidas simplesmente falando o nome da pessoa, e tenha tudo automaticamente anotado em uma planilha do Google Sheets.

> "Alexa, abrir controle de presenca"
> "Joao Silva chegou"
> "Entendido. Joao Silva foi registrado como presente."

## Como Funciona

A skill roda como uma funcao AWS Lambda que recebe comandos de voz da Alexa e escreve os registros em uma planilha do Google Sheets. Cada dia recebe sua propria aba (nomeada pela data), com colunas para **Nome**, **Chegada** e **Saida**.

Uma segunda funcao Lambda e executada diariamente por agendamento para criar a aba do novo dia antes do inicio do expediente.

### Comandos de Voz

| Acao | Exemplos |
|---|---|
| Registrar chegada | "Joao chegou", "registrar chegada de Joao", "marcar Joao como presente" |
| Registrar saida | "Joao saiu", "registrar saida de Joao", "Joao esta saindo" |

Quando uma saida e registrada, a skill encontra a linha de chegada mais recente da pessoa e preenche o horario de saida. Se nao existir uma chegada registrada, ela pede confirmacao antes de registrar uma saida avulsa.

## Arquitetura

```
Dispositivo Alexa  -->  Servico Alexa  -->  Lambda (Go, ARM64)  -->  Google Sheets API
                                                 |
                                            AWS Secrets Manager
                                            (credenciais Google)

EventBridge (cron diario)  -->  Lambda Sheet-Setup  -->  Google Sheets API
```

**Stack tecnologico:** Go, AWS Lambda, AWS CDK (TypeScript), Google Sheets API v4, Alexa Skills Kit

## Pre-requisitos

- **Conta AWS** com permissoes para criar funcoes Lambda, roles IAM e secrets no Secrets Manager
- **Conta de desenvolvedor Amazon** para criar e configurar a skill Alexa
- **Projeto no Google Cloud** com a API do Sheets habilitada
- **Service account do Google** com acesso de Editor na planilha de destino
- **Go 1.25+** para compilar os binarios Lambda
- **Node.js 18+** para deploy via CDK (se usar CDK)
- **GNU Make** ou **PowerShell** (Windows) para executar os comandos de build/deploy

## Configuracao

### 1. Google Sheets

1. Crie uma nova planilha no Google Sheets (ou use uma existente).
2. Crie uma service account no Google Cloud com a API do Sheets habilitada.
3. Compartilhe a planilha com o email da service account (permissao de Editor).
4. Baixe o arquivo JSON de credenciais da service account.

### 2. AWS Secrets Manager

Armazene as credenciais do Google no AWS Secrets Manager:

```bash
make store-secret
# Espera um arquivo credentials.json na raiz do projeto
```

### 3. Skill Alexa

1. Crie uma nova skill custom no [Console de Desenvolvedor Alexa](https://developer.amazon.com/alexa/console/ask).
2. Defina o nome de invocacao como **"controle de presenca"** (ou o nome de sua preferencia).
3. Importe o modelo de interacao de `skill-package/interactionModels/pt-BR.json`.
4. Configure o endpoint da skill com o ARN da funcao Lambda (obtido apos o deploy).

### 4. Deploy

Tres opcoes de deploy estao disponiveis:

#### Opcao A: AWS CDK (recomendado)

```bash
# Compilar binarios Lambda (execute no WSL ou Linux para cross-compilation)
make build-all-artifacts

# Fazer deploy da stack
make deploy-cdk \
  GOOGLE_SHEET_ID=id-da-sua-planilha \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx
```

#### Opcao B: CloudFormation

```bash
make deploy-cfn \
  GOOGLE_SHEET_ID=id-da-sua-planilha \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx \
  S3_BUCKET=seu-bucket-de-deploy
```

#### Opcao C: Lambda CLI direto

```bash
# Criacao inicial
make create-lambda \
  GOOGLE_SHEET_ID=id-da-sua-planilha \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx

# Atualizacoes posteriores
make deploy-lambda
```

#### PowerShell (Windows)

```powershell
.\deploy.ps1 deploy-cdk -GoogleSheetId "id-da-sua-planilha" -AlexaSkillId "amzn1.ask.skill.xxx"
```

### Variaveis de Ambiente

| Variavel | Descricao | Padrao |
|---|---|---|
| `GOOGLE_SHEET_ID` | ID da planilha do Google Sheets | *(obrigatorio)* |
| `ALEXA_SKILL_ID` | ID da Skill Alexa para validacao de requisicoes | *(opcional)* |
| `GOOGLE_CREDENTIALS_SECRET` | Nome do secret no Secrets Manager | `alexa-skill/google-credentials` |
| `TZ` | Fuso horario para os registros de horario | `America/Sao_Paulo` |

| Variavel do Make | Descricao | Padrao |
|---|---|---|
| `REGION` | Regiao AWS | `eu-west-1` |
| `FUNCTION_NAME` | Nome da funcao Lambda | `alexa-attendance-skill` |

## Estrutura do Projeto

```
alexa-attendance-skill/
├── cmd/
│   ├── lambda/          # Entry point da Lambda da skill Alexa
│   └── sheet-setup/     # Lambda de criacao diaria de aba
├── internal/
│   ├── alexa/           # Tratamento de request/response da Alexa
│   ├── sheets/          # Cliente da API do Google Sheets
│   └── awsutil/         # Integracao com AWS Secrets Manager
├── infra/
│   ├── cdk/             # Stack AWS CDK (TypeScript)
│   └── cloudformation/  # Template CloudFormation
├── skill-package/       # Manifesto da skill e modelo de interacao
├── Makefile             # Targets de build e deploy
└── deploy.ps1           # Script de deploy em PowerShell
```

## Testes

```bash
go test ./...
```

A suite de testes cobre o tratamento de requisicoes Alexa (roteamento de intents, estado de sessao, validacao) e operacoes no Google Sheets (criacao de linhas, atualizacoes, setup de abas) usando interfaces mock.

## Licenca

Este projeto e disponibilizado como esta para uso pessoal e educacional.
