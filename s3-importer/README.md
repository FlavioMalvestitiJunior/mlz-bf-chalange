# S3 Importer Service - Guia de Configuração

## Visão Geral

O S3 Importer é um serviço independente que roda como um job agendado (cron) a cada 10 minutos. Ele:
1. Busca templates ativos do banco de dados
2. Baixa JSONs das URLs S3 configuradas
3. Mapeia os campos conforme configuração
4. Publica ofertas no Kafka topic `offers`

## Arquitetura

```
┌─────────────────────┐
│  Cron Job (10 min)  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  S3 Importer        │
│  - Fetch Templates  │
│  - Download JSON    │
│  - Map Fields       │
│  - Produce Kafka    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Kafka (offers)     │
└─────────────────────┘
```

## Configuração

### 1. Agendar Execução

#### Linux/Mac (Crontab)

```bash
# Editar crontab
crontab -e

# Adicionar linha (executar a cada 10 minutos)
*/10 * * * * cd /caminho/para/bf-offers && docker-compose run --rm s3-importer >> /var/log/s3-importer.log 2>&1
```

Ou usar o script fornecido:

```bash
# Dar permissão de execução
chmod +x run-s3-importer.sh

# Adicionar ao crontab
*/10 * * * * /caminho/para/bf-offers/run-s3-importer.sh >> /var/log/s3-importer.log 2>&1
```

#### Windows (Task Scheduler)

1. Abra o **Agendador de Tarefas** (Task Scheduler)
2. Clique em **Criar Tarefa Básica**
3. Nome: "S3 Importer"
4. Gatilho: **Repetir a tarefa a cada: 10 minutos**
5. Ação: **Iniciar um programa**
   - Programa: `C:\Users\Flavio\OneDrive\Desktop\bf-offers\run-s3-importer.bat`
6. Finalizar

### 2. Execução Manual

Para testar ou executar manualmente:

```bash
# Executar uma vez
docker-compose run --rm s3-importer

# Ver logs
docker-compose run --rm s3-importer 2>&1 | tee s3-importer.log
```

### 3. Kubernetes (CronJob)

Se estiver usando Kubernetes:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: s3-importer
spec:
  schedule: "*/10 * * * *"  # A cada 10 minutos
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: s3-importer
            image: bf-offers-s3-importer:latest
            env:
            - name: KAFKA_BROKERS
              value: "kafka:9092"
            - name: KAFKA_OFFERS_TOPIC
              value: "offers"
            - name: POSTGRES_HOST
              value: "postgres"
            - name: POSTGRES_PORT
              value: "5432"
            - name: POSTGRES_USER
              value: "offerbot"
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            - name: POSTGRES_DB
              value: "offerbot"
          restartPolicy: OnFailure
```

## Monitoramento

### Ver Logs

```bash
# Logs do último run
docker-compose logs s3-importer

# Logs em tempo real (se estiver rodando)
docker-compose logs -f s3-importer
```

### Verificar Última Execução

```sql
-- Conectar ao PostgreSQL
docker exec -it postgres psql -U offerbot -d offerbot

-- Ver última execução de cada template
SELECT id, name, is_active, last_run_at 
FROM import_templates 
ORDER BY last_run_at DESC;
```

### Verificar Mensagens no Kafka

```bash
# Ver mensagens produzidas
docker exec -it kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic offers \
  --from-beginning \
  --max-messages 10
```

## Troubleshooting

### Importer não executa

```bash
# Verificar se o serviço está configurado
docker-compose config | grep s3-importer

# Testar execução manual
docker-compose run --rm s3-importer

# Ver logs de erro
docker-compose logs s3-importer
```

### Templates não são processados

```bash
# Verificar templates ativos
docker exec -it postgres psql -U offerbot -d offerbot \
  -c "SELECT * FROM import_templates WHERE is_active = true;"

# Verificar conectividade com S3
docker-compose run --rm s3-importer sh -c "wget -O- 'SUA_URL_S3'"
```

### Mensagens não chegam no Kafka

```bash
# Verificar tópico Kafka
docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092

# Verificar se o tópico 'offers' existe
docker exec -it kafka kafka-topics --describe --topic offers --bootstrap-server localhost:9092

# Ver últimas mensagens
docker exec -it kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic offers \
  --from-beginning \
  --max-messages 5
```

## Variáveis de Ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `KAFKA_BROKERS` | `kafka:9092` | Endereços dos brokers Kafka |
| `KAFKA_OFFERS_TOPIC` | `offers` | Tópico para publicar ofertas |
| `POSTGRES_HOST` | `postgres` | Host do PostgreSQL |
| `POSTGRES_PORT` | `5432` | Porta do PostgreSQL |
| `POSTGRES_USER` | `offerbot` | Usuário do banco |
| `POSTGRES_PASSWORD` | `offerbot123` | Senha do banco |
| `POSTGRES_DB` | `offerbot` | Nome do banco |

## Performance

### Recursos

O serviço é configurado com:
- **CPU Limit**: 0.3 cores
- **Memory Limit**: 64MB
- **CPU Reservation**: 0.05 cores
- **Memory Reservation**: 32MB

### Otimização

Para processar mais templates ou JSONs maiores:

```yaml
# No docker-compose.yml
deploy:
  resources:
    limits:
      cpus: '0.5'
      memory: 128M
```

## Desenvolvimento

### Build Local

```bash
cd s3-importer
go mod download
go build -o s3-importer .
./s3-importer
```

### Testes

```bash
# Criar template de teste via webclient
# http://localhost:8082/import.html

# Executar importer
docker-compose run --rm s3-importer

# Verificar logs
docker-compose logs s3-importer | grep "Produced"
```

## Estrutura do Serviço

```
s3-importer/
├── internal/
│   ├── models/
│   │   └── models.go          # Modelos de dados
│   ├── repository/
│   │   └── import_template_repository.go  # Acesso ao DB
│   └── importer/
│       └── s3_importer.go     # Lógica de importação
├── main.go                    # Entry point
├── go.mod                     # Dependências
├── go.sum
└── Dockerfile                 # Build Docker
```

## Próximos Passos

1. **Alertas**: Configurar alertas se o job falhar
2. **Métricas**: Adicionar Prometheus metrics
3. **Retry**: Implementar retry automático em caso de falha
4. **Paralelização**: Processar múltiplos templates em paralelo
5. **Validação**: Adicionar validação de schema JSON
