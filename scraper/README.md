# Promobit Scraper - Atualização para API

## Mudanças Implementadas

### Antes (HTML Scraping)
- Usava Colly para fazer scraping do HTML
- Seletores CSS frágeis (`.pr-card-item`, `.pr-title`, etc.)
- Lento e sujeito a quebrar com mudanças no site

### Depois (API Calls)
- Usa APIs oficiais do Promobit
- Dados estruturados em JSON
- Mais rápido e confiável
- Paginação automática

## APIs Utilizadas

### 1. Search API (Busca com Paginação)
**Endpoint**: `https://api.promobit.com.br/search/result/offers?q={query}&page={page}`

**Exemplo**:
```
https://api.promobit.com.br/search/result/offers?q=iphone&page=1
```

**Resposta**:
```json
{
  "data": {
    "offers": [
      {
        "id": 12345,
        "title": "iPhone 15 Pro Max",
        "price": 7200.00,
        "old_price": 8999.00,
        "description": "256GB, Titânio Natural",
        "url": "https://...",
        "is_active": true,
        "cashback": {
          "percentage": 5
        }
      }
    ],
    "meta": {
      "current_page": 1,
      "last_page": 3
    }
  }
}
```

**Features**:
- ✅ Paginação automática até `last_page`
- ✅ Filtra apenas ofertas ativas (`is_active: true`)
- ✅ Delay de 1 segundo entre páginas

### 2. Home API (Next.js Data)
**Endpoint**: `https://www.promobit.com.br/_next/data/bcc3e837c1/index.json`

**Resposta**:
```json
{
  "pageProps": {
    "offers": [
      {
        "id": 12345,
        "title": "...",
        "price": 7200.00,
        "old_price": 8999.00,
        "is_active": true,
        ...
      }
    ]
  }
}
```

**Features**:
- ✅ Ofertas da home page
- ✅ Filtra apenas ofertas ativas

## Fluxo de Dados

### 1. Scraping Periódico (Home)
```
Timer (5 min)
    ↓
GET /index.json
    ↓
Filter is_active = true
    ↓
Convert to Offer
    ↓
Kafka (offers topic)
```

### 2. Scraping por Busca (Wishlist)
```
Wishlist Event
    ↓
GET /search/result/offers?q=produto&page=1
    ↓
Loop through all pages
    ↓
Filter is_active = true
    ↓
Convert to Offer
    ↓
Kafka (offers topic)
```

## Código Principal

### Busca com Paginação
```go
func scrapePromobitSearch(producer sarama.SyncProducer, config Config, query string) {
    encodedQuery := url.QueryEscape(query)
    page := 1
    
    for {
        apiURL := fmt.Sprintf("https://api.promobit.com.br/search/result/offers?q=%s&page=%d", 
            encodedQuery, page)
        
        // Fetch page
        resp, err := http.Get(apiURL)
        // ... handle response
        
        // Parse JSON
        var searchResp PromobitSearchResponse
        json.Unmarshal(body, &searchResp)
        
        // Process active offers
        for _, offer := range searchResp.Data.Offers {
            if !offer.IsActive {
                continue
            }
            publishOffer(producer, convertPromobitOffer(offer), config.KafkaOffersTopic)
        }
        
        // Check if more pages
        if page >= searchResp.Data.Meta.LastPage {
            break
        }
        
        page++
        time.Sleep(1 * time.Second) // Polite delay
    }
}
```

### Home Page
```go
func scrapePromobitHome(producer sarama.SyncProducer, config Config) {
    resp, err := http.Get("https://www.promobit.com.br/_next/data/bcc3e837c1/index.json")
    
    var homeResp PromobitHomeResponse
    json.Unmarshal(body, &homeResp)
    
    for _, offer := range homeResp.PageProps.Offers {
        if !offer.IsActive {
            continue
        }
        publishOffer(producer, convertPromobitOffer(offer), config.KafkaOffersTopic)
    }
}
```

## Vantagens

### Performance
- ✅ Mais rápido (JSON vs HTML parsing)
- ✅ Menos uso de memória
- ✅ Sem dependência do Colly

### Confiabilidade
- ✅ API estável (não quebra com mudanças no site)
- ✅ Dados estruturados
- ✅ Menos erros de parsing

### Features
- ✅ Paginação automática
- ✅ Filtra apenas ofertas ativas
- ✅ Cashback incluído
- ✅ Preço original (oldPrice)

## Estruturas de Dados

```go
type PromobitSearchResponse struct {
    Data struct {
        Offers []PromobitOffer `json:"offers"`
        Meta   struct {
            CurrentPage int `json:"current_page"`
            LastPage    int `json:"last_page"`
        } `json:"meta"`
    } `json:"data"`
}

type PromobitOffer struct {
    ID          int     `json:"id"`
    Title       string  `json:"title"`
    Price       float64 `json:"price"`
    OldPrice    float64 `json:"old_price"`
    Description string  `json:"description"`
    URL         string  `json:"url"`
    IsActive    bool    `json:"is_active"`
    Cashback    struct {
        Percentage int `json:"percentage"`
    } `json:"cashback"`
}
```

## Teste

### Teste Manual

```bash
# Build
docker-compose build scraper

# Run
docker-compose up scraper

# Ver logs
docker-compose logs -f scraper
```

### Teste de Busca

```bash
# Adicionar item à wishlist via Telegram
/add iPhone 15 R$7000

# O scraper irá:
# 1. Receber evento do Kafka
# 2. Buscar "iPhone 15" na API
# 3. Paginar por todas as páginas
# 4. Publicar ofertas ativas no Kafka
```

### Verificar Mensagens no Kafka

```bash
docker exec -it kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic offers \
  --from-beginning \
  --max-messages 10
```

## Monitoramento

### Logs Importantes

```
Fetching Promobit Home via API...
Published 15 offers from Promobit home

Searching Promobit API for: iphone
Published 20 active offers from page 1
Published 18 active offers from page 2
Published 12 active offers from page 3
Total published 50 active offers for query: iphone
```

### Métricas

- Ofertas publicadas por execução
- Páginas processadas
- Ofertas ativas vs inativas
- Tempo de resposta da API

## Troubleshooting

### API retorna erro

```bash
# Verificar conectividade
docker-compose run --rm scraper sh -c "wget -O- 'https://api.promobit.com.br/search/result/offers?q=iphone&page=1'"

# Verificar logs
docker-compose logs scraper | grep "Failed"
```

### Nenhuma oferta publicada

```bash
# Verificar se há ofertas ativas
curl "https://api.promobit.com.br/search/result/offers?q=iphone&page=1" | jq '.data.offers[] | select(.is_active == true)'

# Verificar logs do scraper
docker-compose logs scraper | grep "Published"
```

### Build ID do Next.js mudou

Se o endpoint `/_next/data/bcc3e837c1/index.json` parar de funcionar, o build ID pode ter mudado.

**Solução**:
1. Acesse https://www.promobit.com.br
2. Abra DevTools → Network
3. Procure por `index.json`
4. Copie o novo build ID
5. Atualize no código

## Dependências Removidas

- ❌ `github.com/gocolly/colly/v2` - Não é mais necessário

## Próximos Passos

1. **Rate Limiting**: Adicionar controle de taxa de requisições
2. **Retry Logic**: Retry automático em caso de falha da API
3. **Cache**: Cachear ofertas já processadas (evitar duplicatas)
4. **Métricas**: Prometheus metrics para monitoramento
5. **Alertas**: Alertas se a API ficar indisponível
