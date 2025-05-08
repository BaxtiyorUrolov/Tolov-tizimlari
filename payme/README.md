# Tolov-tizimlari

Bu loyiha O‘zbekistondagi to‘lov tizimlari bilan ishlash uchun Go package’lar to‘plamini taqdim etadi. Hozirda faqat Payme to‘lov tizimi qo‘llab-quvvatlanadi.

## O‘rnatish

Loyihani o‘rnatish uchun quyidagi buyruqni ishlatishingiz mumkin:

```bash
go get github.com/myusername/Tolov-tizimlari/payme
```

## Talablar

- PostgreSQL ma'lumotlar bazasi
- Payme API kaliti (PAYME_KEY)
- Payme Merchant ID

## Konfiguratsiya

Loyihadan foydalanish uchun `.env` faylida quyidagi o‘zgaruvchilarni sozlang.

```plaintext
DATABASE_URL=host=localhost user=postgres dbname=yourdb sslmode=disable
PAYME_KEY=your-payme-key-here
PAYME_BASE_URL=https://test.paycom.uz
PAYME_MERCHANT_ID=your-merchant-id-here
```

## Foydalanish

### Payme tranzaksiyasini yaratish
Payme to‘lov tizimi bilan ishlash uchun quyidagi misoldan foydalaning:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/myusername/Tolov-tizimlari/payme"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database connection
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// Create payme handler
	handler := payme.NewHandler(db, os.Getenv("PAYME_KEY"), os.Getenv("PAYME_BASE_URL"))

	// Create a transaction
	userID := 123
	amount := 100
	paymeID := os.Getenv("PAYME_MERCHANT_ID")
	returnURL := "http://example.com/callback"

	link, err := handler.CreatePaymeTransaction(userID, amount, paymeID, returnURL)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println("payme transaction link:", link)
}
```

### Webhook so‘rovlarini qayta ishlash
Webhook’lar bilan ishlash uchun `payme/cmd/main.go` faylidagi misolni ko‘ring.

## Ma'lumotlar bazasi sxemasi
Payme tranzaksiyalarni saqlash uchun quyidagi SQL sxemasidan foydalaning:

```sql
CREATE TABLE payme (
                       id TEXT PRIMARY KEY,
                       payme_transaction_id TEXT,
                       user_id INTEGER,
                       amount INTEGER,
                       state INTEGER,
                       create_time TIMESTAMP,
                       perform_time TIMESTAMP,
                       cancel_time TIMESTAMP
);
```

## Litsenziya
MIT