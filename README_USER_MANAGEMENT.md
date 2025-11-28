# User Management Features - Webclient Dashboard

## 📋 Funcionalidades Implementadas

### 1. Busca de Usuários
- Campo de busca em tempo real com debounce de 300ms
- Busca por nome, sobrenome ou username
- Resultados exibidos na mesma tabela de usuários ativos

### 2. Visualização de Wishlist
- Clique no botão 📋 para ver a lista de desejos do usuário
- Modal exibe todos os itens com:
  - Nome do produto
  - Preço alvo
  - Porcentagem de desconto
  - Data de criação
- Cache Redis de 5 minutos para performance

### 3. Gerenciamento de Usuários
- **Blacklist**: Botão 🚫 para adicionar à blacklist
- **Unblacklist**: Botão ✅ para remover da blacklist
- **Delete**: Botão 🗑️ para deletar usuário permanentemente
- Todas as ações requerem confirmação
- Operações transacionais no PostgreSQL
- Sincronização automática com Redis

## 🗄️ Arquitetura de Dados

### PostgreSQL (Persistência)
- Tabela `users` com coluna `is_blacklisted`
- Tabela `wishlists` com relação ao usuário
- Operações transacionais para garantir consistência

### Redis (Cache)
- `wishlist:{user_id}` - Cache de wishlists (TTL: 5 minutos)
- `blacklist:{user_id}` - Marcador de usuários blacklistados
- Invalidação automática em operações de delete

## 🚀 Deploy

### 1. Executar Migration
Antes de fazer deploy, execute a migration SQL:

```bash
docker-compose exec postgres psql -U offerbot -d offerbot -f /docker-entrypoint-initdb.d/migration_add_blacklist.sql
```

Ou execute manualmente:
```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_blacklisted BOOLEAN DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_users_blacklisted ON users(is_blacklisted);
```

### 2. Rebuild e Start
```bash
docker-compose up --build webclient
```

### 3. Acessar Dashboard
Navegue para: `http://localhost:8082`

## 🔌 API Endpoints

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/api/users/search?q={query}` | Busca usuários |
| GET | `/api/users/{id}/wishlist` | Retorna wishlist do usuário |
| POST | `/api/users/{id}/blacklist` | Adiciona à blacklist |
| DELETE | `/api/users/{id}/blacklist` | Remove da blacklist |
| DELETE | `/api/users/{id}` | Deleta usuário |

## 📁 Arquivos Modificados

### Backend
- `webclient/internal/repository/stats_repository.go` - Lógica de dados
- `webclient/internal/handlers/dashboard.go` - Handlers HTTP
- `webclient/internal/models/models.go` - Modelos de dados
- `webclient/main.go` - Inicialização Redis e rotas

### Frontend
- `webclient/static/index.html` - UI (search box + modal)
- `webclient/static/js/dashboard.js` - Lógica JavaScript
- `webclient/static/css/style.css` - Estilos

### Database
- `migration_add_blacklist.sql` - Migration para coluna is_blacklisted

## 🎨 Interface do Usuário

### Tabela de Usuários
```
┌─────────────┬──────────┬──────────┬────────┬──────────────┬────────┐
│ Telegram ID │ Nome     │ Username │ Listas │ Última Ativ. │ Ações  │
├─────────────┼──────────┼──────────┼────────┼──────────────┼────────┤
│ 123456789   │ João     │ @joao    │ 5      │ 2h atrás     │ 📋🚫🗑️ │
└─────────────┴──────────┴──────────┴────────┴──────────────┴────────┘
```

### Ações Disponíveis
- 📋 **Ver Wishlist** - Abre modal com lista de desejos
- 🚫 **Blacklist** - Adiciona usuário à blacklist
- ✅ **Unblacklist** - Remove da blacklist (aparece quando blacklistado)
- 🗑️ **Delete** - Deleta usuário e todos os dados

## ⚠️ Notas Importantes

1. **Operações Destrutivas**: Delete é permanente e remove todos os dados do usuário
2. **Cache**: Wishlists são cacheadas por 5 minutos no Redis
3. **Transações**: Todas as operações de delete são transacionais
4. **Confirmações**: Todas as ações destrutivas requerem confirmação do usuário

## 🔧 Variáveis de Ambiente

O webclient usa as seguintes variáveis (já configuradas no docker-compose.yml):

```env
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=offerbot
POSTGRES_PASSWORD=offerbot123
POSTGRES_DB=offerbot
```

## ✅ Checklist de Verificação

- [x] Migration executada no banco de dados
- [x] Redis está rodando e acessível
- [x] Webclient rebuilded com as novas dependências
- [x] Dashboard acessível em localhost:8082
- [x] Busca de usuários funcionando
- [x] Modal de wishlist abrindo corretamente
- [x] Ações de blacklist/delete funcionando
- [x] Confirmações aparecendo antes de ações destrutivas
