# Exemplo de mensagem SNS para teste

Publique esta mensagem na sua fila SNS para testar o sistema:

```json
{
  "product_name": "iPhone 15 Pro Max 256GB",
  "price": 6500.00,
  "original_price": 9999.00,
  "discount_percentage": 35,
  "cashback_percentage": 5,
  "source": "Amazon Black Friday"
}
```

## Usando AWS CLI

```bash
aws sns publish \
  --topic-arn arn:aws:sns:us-east-1:123456789012:offers-topic \
  --message '{
    "product_name": "iPhone 15 Pro Max 256GB",
    "price": 6500.00,
    "original_price": 9999.00,
    "discount_percentage": 35,
    "cashback_percentage": 5,
    "source": "Amazon Black Friday"
  }'
```

## Mais exemplos

### Oferta de TV
```json
{
  "product_name": "Smart TV Samsung 55\" 4K",
  "price": 2199.00,
  "original_price": 3499.00,
  "discount_percentage": 37,
  "cashback_percentage": 10,
  "source": "Magazine Luiza"
}
```

### Oferta de Notebook
```json
{
  "product_name": "Notebook Dell Inspiron 15 i5 8GB",
  "price": 2999.00,
  "original_price": 4299.00,
  "discount_percentage": 30,
  "cashback_percentage": 0,
  "source": "Kabum"
}
```

### Oferta de Fone
```json
{
  "product_name": "Fone Bluetooth Sony WH-1000XM5",
  "price": 1499.00,
  "original_price": 2199.00,
  "discount_percentage": 32,
  "cashback_percentage": 15,
  "source": "Fast Shop"
}
```
