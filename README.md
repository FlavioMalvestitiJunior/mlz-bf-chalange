# Telegram Offer Monitor Bot 🤖

Sistema distribuído de monitoramento de ofertas e cashbacks via Telegram Bot, com integração SNS, comunicação via Kafka, e persistência dupla (Redis + PostgreSQL).

## 📋 Índice

- [Arquitetura](#-arquitetura)
- [Funcionalidades](#-funcionalidades)
- [Pré-requisitos](#-pré-requisitos)
- [Configuração](#-configuração)
- [Instalação](#-instalação)
- [Uso](#-uso)
- [Comandos do Bot](#-comandos-do-bot)
- [Escalabilidade](#-escalabilidade)
- [Estrutura do Projeto](#-estrutura-do-projeto)

## 🏗 Arquitetura

```
┌─────────────┐       ┌─────────────┐
│  AWS SNS    │       │   S3 JSON   │
│   Queue     │       │   Files     │
└──────┬──────┘       └──────┬──────┘
       │                     │
       │                     │ (HTTP GET)
       ▼                     ▼
┌─────────────────────────────────────┐
│       Backend Service (Go)          │
│  - SNS Consumer                     │
│  - S3 Import Scheduler (10 min)    │
│  - Offer Matcher                    │
│  - Kafka Producer                   │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐
│   Kafka     │
└──────┬──────┘
       │
       ├──────────────────┐
       ▼                  ▼
┌─────────────────┐  ┌─────────────────┐
│  Frontend       │  │  Webclient      │
│  (Telegram Bot) │  │  (Dashboard)    │
│  - Kafka        │  │  - Import UI    │
│    Consumer     │  │  - Stats        │
│  - Bot Handlers │  │  - Templates    │
└─────────────────┘  └─────────────────┘
       │                  │
       ▼                  ▼
┌─────────────┐    ┌─────────────┐
│  Telegram   │    │   Browser   │
│    Users    │    │   (Port     │
└─────────────┘    │    8082)    │
                   └─────────────┘

┌─────────────┐  ┌──────────────┐
│   Redis     │  │  PostgreSQL  │
│  (Cache)    │  │ (Persistence)│
└─────────────┘  └──────────────┘
```

## ✨ Funcionalidades

### Bot Telegram
- 🔍 **Monitoramento 24/7** de ofertas via fila SNS
- 💰 **Alertas por preço** - Notifica quando produto atinge preço desejado
- 🔥 **Alertas por desconto** - Notifica quando desconto atinge percentual mínimo
- 📱 **Interface Telegram** - Gerenciamento completo via bot

### Importação S3
- 📥 **Importação automática** de ofertas de arquivos JSON no S3
- 🗺️ **Mapeamento flexível** de campos JSON para modelo interno
- ⏰ **Scheduler** executa importações a cada 10 minutos
- 🎯 **Suporte a JSON paths** - mapeia campos aninhados (ex: `data.product.name`)

### Dashboard Web
- 📊 **Dashboard de estatísticas** - visualize métricas do sistema
- 🛠️ **Gerenciamento de templates** - configure importações S3 via interface web
- 📋 **Templates de mensagens** - personalize notificações

### Infraestrutura
- 💾 **Persistência dupla** - Redis para cache + PostgreSQL para dados permanentes
- 🔄 **Auto-recuperação** - Sistema se recupera após reinicialização de pods
- 📊 **Escalável** - Backend pode rodar em múltiplas instâncias
- ⚡ **Otimizado** - Mínimo uso de recursos por pod

## 📦 Pré-requisitos

- Docker & Docker Compose
- Conta AWS (para SNS) ou LocalStack para desenvolvimento
- Token do Telegram Bot (obtenha via [@BotFather](https://t.me/botfather))

### Como criar um Telegram Bot

1. Abra o Telegram e procure por [@BotFather](https://t.me/botfather)
2. Envie `/newbot`
3. Siga as instruções para escolher nome e username
4. Copie o token fornecido

## ⚙️ Configuração

1. **Clone o repositório**
```bash
cd bf-offers
```

2. **Configure as variáveis de ambiente**
```bash
cp .env.example .env
```

3. **Edite o arquivo `.env`** com suas credenciais:

```env
# Telegram Bot Token (OBRIGATÓRIO)
TELEGRAM_BOT_TOKEN=seu_token_aqui

# AWS SNS (configure conforme seu ambiente)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=sua_access_key
AWS_SECRET_ACCESS_KEY=sua_secret_key
SNS_QUEUE_URL=sua_queue_url

# Outras configurações já vêm com valores padrão
```

## 🚀 Instalação

### Opção 1: Iniciar todos os serviços

```bash
docker-compose up -d
```

### Opção 2: Build e iniciar

```bash
docker-compose build
docker-compose up -d
```

### Verificar status dos serviços

```bash
docker-compose ps
```

### Ver logs

```bash
# Todos os serviços
docker-compose logs -f

# Backend apenas
docker-compose logs -f backend

# Frontend apenas
docker-compose logs -f frontend

# Webclient apenas
docker-compose logs -f webclient
```

## 📱 Uso

### Comandos do Bot

Abra o Telegram e procure pelo seu bot. Comandos disponíveis:

#### `/start`
Inicia o bot e mostra mensagem de boas-vindas

#### `/add <produto> <preço|desconto%>`
Adiciona produto à lista de desejos

**Exemplos:**
```
/add iPhone 15 R$4000
/add Samsung TV 30%
/add Notebook Gamer 25%
/add Fone Bluetooth 150
```

#### `/list`
Lista todos os produtos na sua lista de desejos

#### `/delete <id>`
Remove produto da lista (use `/list` para ver os IDs)

**Exemplo:**
```
/delete 1
```

#### `/help`
Mostra ajuda com todos os comandos

### Fluxo de Uso

1. **Adicione produtos à lista:**
   ```
   /add iPhone 15 R$4500
   /add Smart TV 40%
   ```

2. **Verifique sua lista:**
   ```
   /list
   ```

3. **Aguarde as notificações!** 🎉
   - O sistema monitora ofertas continuamente
   - Você receberá uma mensagem quando uma oferta corresponder aos seus critérios

4. **Gerencie sua lista:**
   ```
   /delete 1
   ```

## 🌐 Dashboard Web

### Acessar o Dashboard

Abra seu navegador em: **http://localhost:8082**

### Funcionalidades do Dashboard

#### 📊 Estatísticas
- Visualize métricas do sistema
- Usuários ativos
- Ofertas processadas

#### 📥 Importação S3

Acesse: **http://localhost:8082/import.html**

**Criar Template de Importação:**

1. **Nome do Template**: Identifique sua fonte de dados
2. **URL do S3**: Cole a URL do arquivo JSON (pública ou pre-signed)
3. **Testar URL**: Clique para visualizar o JSON e obter sugestões de mapeamento
4. **Mapear Campos**:
   - `ProductName` (obrigatório) - ex: `titulo`, `product.name`
   - `Price` - ex: `price`, `pricing.current`
   - `OriginalPrice` - ex: `oldPrice`, `pricing.original`
   - `Details` - ex: `details`, `description`
   - `CashbackPercentage` - ex: `percentCashback`
   - `Source` - ex: `source`, `provider`
5. **Ativar**: Marque para executar a cada 10 minutos
6. **Salvar**: Template será executado automaticamente

**Exemplo de JSON Suportado:**
```json
[
  {
    "titulo": "iPhone 15 Pro",
    "price": 7200.00,
    "oldPrice": 8999.00,
    "details": "128GB, Titânio Azul",
    "percentCashback": 5,
    "source": "Amazon"
  }
]
```

**Gerenciar Templates:**
- ✏️ Editar templates existentes
- 🔄 Ativar/Desativar importações
- 🗑️ Excluir templates
- 📅 Ver última execução

#### 📋 Templates de Mensagens

Acesse: **http://localhost:8082/templates.html**

Personalize as mensagens enviadas aos usuários.

## 📈 Escalabilidade

### Escalar o Backend

O backend pode rodar em múltiplas instâncias para processar mais ofertas:

```bash
# Escalar para 3 instâncias
docker-compose up -d --scale backend=3

# Verificar instâncias
docker-compose ps backend
```

### Monitorar Recursos

```bash
# Ver uso de CPU e memória
docker stats

# Ver logs de uma instância específica
docker logs bf-offers-backend-1
```

### Configurações de Recursos

No `docker-compose.yml`, cada serviço está configurado com:
- **Limite**: 128MB RAM, 0.5 CPU
- **Reserva**: 64MB RAM, 0.1 CPU

Ajuste conforme necessário para seu volume de ofertas.

## 📁 Estrutura do Projeto

```
bf-offers/
├── backend/                    # Backend Service (Go)
│   ├── internal/
│   │   ├── consumer/          # SNS Consumer
│   │   ├── matcher/           # Offer Matching Logic
│   │   ├── producer/          # Kafka Producer
│   │   ├── repository/        # Data Access Layer
│   │   └── models/            # Data Models
│   ├── main.go                # Entry Point
│   ├── Dockerfile             # Docker Build
│   └── go.mod                 # Dependencies
│
├── frontend/                   # Frontend Service (Go)
│   ├── internal/
│   │   ├── bot/               # Telegram Bot Handlers
│   │   ├── consumer/          # Kafka Consumer
│   │   ├── repository/        # Data Access Layer
│   │   └── models/            # Data Models
│   ├── main.go                # Entry Point
│   ├── Dockerfile             # Docker Build
│   └── go.mod                 # Dependencies
│
├── webclient/                  # Webclient Service (Go)
│   ├── internal/
│   │   ├── handlers/          # HTTP Handlers
│   │   ├── repository/        # Data Access Layer
│   │   └── models/            # Data Models
│   ├── static/                # Static Files (HTML/CSS/JS)
│   │   ├── import.html        # S3 Import UI
│   │   ├── templates.html     # Message Templates UI
│   │   └── js/                # JavaScript
│   ├── main.go                # Entry Point
│   ├── Dockerfile             # Docker Build
│   └── go.mod                 # Dependencies
│
├── docker-compose.yml          # Orchestration
├── init.sql                    # Database Schema
├── .env.example                # Environment Template
└── README.md                   # This file
```

## 🔧 Desenvolvimento

### Testar localmente sem Docker

**Backend:**
```bash
cd backend
go mod download
export $(cat ../.env | xargs)
go run main.go
```

**Frontend:**
```bash
cd frontend
go mod download
export $(cat ../.env | xargs)
go run main.go
```

### Formato de Mensagem SNS

O backend espera mensagens no seguinte formato JSON:

```json
{
  "product_name": "iPhone 15",
  "price": 4200.00,
  "original_price": 5999.00,
  "discount_percentage": 30,
  "cashback_percentage": 5,
  "source": "Amazon"
}
```

### Testar com mensagem de exemplo

Publique uma mensagem de teste na sua fila SNS:

```bash
aws sns publish \
  --topic-arn seu-topic-arn \
  --message '{"product_name":"iPhone 15","price":4000,"original_price":5999,"discount_percentage":33,"cashback_percentage":5,"source":"Test"}'
```

## 🛠 Troubleshooting

### Bot não responde
1. Verifique se o token está correto no `.env`
2. Verifique logs: `docker-compose logs frontend`
3. Certifique-se de que o Kafka está rodando: `docker-compose ps kafka`

### Backend não processa ofertas
1. Verifique credenciais AWS no `.env`
2. Verifique logs: `docker-compose logs backend`
3. Teste conectividade com SNS

### Banco de dados não conecta
1. Aguarde alguns segundos após `docker-compose up` (health checks)
2. Verifique: `docker-compose logs postgres`
3. Reinicie: `docker-compose restart`

### Limpar tudo e recomeçar
```bash
docker-compose down -v
docker-compose up -d
```

## 📊 Monitoramento

### Health Checks

- Backend: `http://localhost:8080/health`
- Frontend: `http://localhost:8081/health`
- Webclient: `http://localhost:8082/health`

### Verificar serviços

```bash
# Status de todos os containers
docker-compose ps

# Logs em tempo real
docker-compose logs -f

# Uso de recursos
docker stats
```

## 🤝 Contribuindo

Sinta-se à vontade para abrir issues ou pull requests!

## 📄 Licença

MIT License

---

**Desenvolvido com ❤️ usando Go, Telegram Bot API, Kafka, Redis e PostgreSQL**
