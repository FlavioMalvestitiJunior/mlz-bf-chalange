# 🚀 Deployment Checklist - User Management Features

## Pré-Deploy

### 1. Verificar Dependências
```bash
# Verificar se Redis está no docker-compose.yml
grep -A 5 "redis:" docker-compose.yml

# Verificar se webclient depende do Redis
grep -A 10 "webclient:" docker-compose.yml
```

### 2. Executar Migration
```bash
# Copiar migration para o container
docker cp migration_add_blacklist.sql postgres:/tmp/

# Executar migration
docker-compose exec postgres psql -U offerbot -d offerbot -f /tmp/migration_add_blacklist.sql

# Verificar se a coluna foi criada
docker-compose exec postgres psql -U offerbot -d offerbot -c "\d users"
```

## Deploy

### 3. Rebuild Services
```bash
# Parar serviços
docker-compose down

# Rebuild apenas o webclient
docker-compose build webclient

# Subir todos os serviços
docker-compose up -d
```

### 4. Verificar Logs
```bash
# Verificar se webclient iniciou corretamente
docker-compose logs -f webclient

# Procurar por:
# - "Connected to database successfully"
# - "Web client starting on port 8082"
```

## Pós-Deploy

### 5. Testes Funcionais

#### Teste 1: Busca de Usuários
- [ ] Acessar http://localhost:8082
- [ ] Digitar nome no campo de busca
- [ ] Verificar se resultados aparecem

#### Teste 2: Visualizar Wishlist
- [ ] Clicar no botão 📋 de um usuário
- [ ] Verificar se modal abre
- [ ] Verificar se itens são exibidos
- [ ] Fechar modal clicando no X ou fora

#### Teste 3: Blacklist
- [ ] Clicar no botão 🚫
- [ ] Confirmar ação
- [ ] Verificar se botão muda para ✅
- [ ] Clicar em ✅ para remover da blacklist

#### Teste 4: Delete
- [ ] Clicar no botão 🗑️
- [ ] Confirmar ação
- [ ] Verificar se usuário desaparece da lista

### 6. Verificar Redis
```bash
# Conectar ao Redis
docker-compose exec redis redis-cli

# Verificar chaves de wishlist
KEYS wishlist:*

# Verificar chaves de blacklist
KEYS blacklist:*

# Verificar TTL de uma wishlist
TTL wishlist:123456789
```

### 7. Verificar PostgreSQL
```bash
# Conectar ao PostgreSQL
docker-compose exec postgres psql -U offerbot -d offerbot

# Verificar usuários blacklistados
SELECT telegram_id, username, is_blacklisted FROM users WHERE is_blacklisted = true;

# Verificar wishlists
SELECT u.username, COUNT(w.id) as wishlists 
FROM users u 
LEFT JOIN wishlists w ON u.telegram_id = w.telegram_id 
GROUP BY u.username;
```

## Rollback (Se Necessário)

### Reverter Migration
```sql
ALTER TABLE users DROP COLUMN IF EXISTS is_blacklisted;
DROP INDEX IF EXISTS idx_users_blacklisted;
```

### Reverter Código
```bash
# Fazer checkout da versão anterior
git checkout <commit-anterior>

# Rebuild
docker-compose build webclient
docker-compose up -d webclient
```

## Monitoramento

### Métricas para Observar
- [ ] Tempo de resposta da busca de usuários
- [ ] Taxa de cache hit no Redis (wishlists)
- [ ] Número de operações de blacklist/delete
- [ ] Erros nos logs do webclient

### Comandos Úteis
```bash
# Ver uso de memória do Redis
docker stats redis

# Ver conexões ativas no PostgreSQL
docker-compose exec postgres psql -U offerbot -d offerbot -c "SELECT count(*) FROM pg_stat_activity;"

# Limpar cache do Redis (se necessário)
docker-compose exec redis redis-cli FLUSHDB
```

## ✅ Checklist Final

- [ ] Migration executada com sucesso
- [ ] Webclient rebuilded e rodando
- [ ] Busca de usuários funcionando
- [ ] Modal de wishlist funcionando
- [ ] Blacklist/Unblacklist funcionando
- [ ] Delete funcionando
- [ ] Redis cacheando wishlists
- [ ] Logs sem erros
- [ ] Performance aceitável
- [ ] Documentação atualizada

## 🆘 Troubleshooting

### Problema: Webclient não inicia
```bash
# Verificar logs
docker-compose logs webclient

# Verificar se Redis está acessível
docker-compose exec webclient ping redis

# Verificar variáveis de ambiente
docker-compose exec webclient env | grep REDIS
```

### Problema: Busca não retorna resultados
```sql
-- Verificar se há usuários no banco
SELECT COUNT(*) FROM users;

-- Verificar dados dos usuários
SELECT telegram_id, username, first_name, last_name FROM users LIMIT 5;
```

### Problema: Modal de wishlist vazio
```bash
# Verificar se há wishlists no banco
docker-compose exec postgres psql -U offerbot -d offerbot -c "SELECT COUNT(*) FROM wishlists;"

# Verificar cache no Redis
docker-compose exec redis redis-cli KEYS "wishlist:*"
```

### Problema: Redis não está cacheando
```bash
# Verificar conexão com Redis
docker-compose exec webclient nc -zv redis 6379

# Verificar logs do Redis
docker-compose logs redis

# Testar manualmente
docker-compose exec redis redis-cli SET test "value"
docker-compose exec redis redis-cli GET test
```
