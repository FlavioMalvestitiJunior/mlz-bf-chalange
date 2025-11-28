package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/bf-offers/frontend/internal/models"
)

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	kafkaProducer sarama.SyncProducer
	commandTopic  string
}

func NewBotHandler(bot *tgbotapi.BotAPI, kafkaProducer sarama.SyncProducer, commandTopic string) *BotHandler {
	return &BotHandler{
		bot:           bot,
		kafkaProducer: kafkaProducer,
		commandTopic:  commandTopic,
	}
}

// HandleUpdate handles incoming Telegram updates
func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	// Handle commands
	if update.Message.IsCommand() {
		h.handleCommand(update.Message)
		return
	}

	// Handle regular messages
	h.sendMessage(update.Message.Chat.ID, "Use /help para ver os comandos disponíveis.")
}

// handleCommand handles bot commands
func (h *BotHandler) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		h.handleStart(message)
	case "help":
		h.handleHelp(message)
	case "add":
		h.handleAdd(message)
	case "list":
		h.handleList(message)
	case "delete", "del":
		h.handleDelete(message)
	default:
		h.sendMessage(message.Chat.ID, "Comando não reconhecido. Use /help para ver os comandos disponíveis.")
	}
}

// handleStart handles the /start command
func (h *BotHandler) handleStart(message *tgbotapi.Message) {
	text := `🎉 *Bem-vindo ao Bot de Ofertas!*

Eu vou te ajudar a monitorar ofertas e cashbacks de produtos!

*Como funciona:*
1️⃣ Adicione produtos à sua lista de desejos
2️⃣ Defina um preço desejado ou desconto mínimo
3️⃣ Receba notificações quando encontrarmos ofertas!

*Comandos disponíveis:*
/add - Adicionar produto à lista
/list - Ver sua lista de desejos
/delete - Remover produto da lista
/help - Ver esta mensagem

*Exemplos:*
` + "`/add iPhone 15 R$4000`" + `
` + "`/add Samsung TV 30%`" + `

Vamos começar? Use /add para adicionar seu primeiro produto! 🚀`

	h.sendMessage(message.Chat.ID, text)
	
	// Send user registration command to backend via Kafka
	h.sendCommandToBackend(models.Command{
		Type:       "register_user",
		TelegramID: message.From.ID,
		Username:   message.From.UserName,
		FirstName:  message.From.FirstName,
		LastName:   message.From.LastName,
	})
}

// handleHelp handles the /help command
func (h *BotHandler) handleHelp(message *tgbotapi.Message) {
	text := `📚 *Ajuda - Comandos Disponíveis*

*Adicionar produto:*
` + "`/add <produto> <preço|desconto%>`" + `

Exemplos:
` + "`/add iPhone 15 R$4000`" + ` - Notifica quando preço ≤ R$4000
` + "`/add Samsung TV 30%`" + ` - Notifica quando desconto ≥ 30%
` + "`/add Notebook Gamer 25%`" + ` - Notifica quando desconto ≥ 25%

*Listar produtos:*
` + "`/list`" + ` - Mostra todos os produtos na sua lista

*Remover produto:*
` + "`/delete <id>`" + ` - Remove produto pelo ID (veja o ID com /list)

Exemplo:
` + "`/delete 1`" + ` - Remove o produto com ID 1

*Dicas:*
• Você pode adicionar quantos produtos quiser
• Use nomes descritivos para facilitar a busca
• O bot monitora ofertas 24/7! 🔍`

	h.sendMessage(message.Chat.ID, text)
}

// handleAdd handles the /add command
func (h *BotHandler) handleAdd(message *tgbotapi.Message) {
	args := message.CommandArguments()
	if args == "" {
		h.sendMessage(message.Chat.ID, "❌ Uso incorreto!\n\nExemplos:\n`/add iPhone 15 R$4000`\n`/add Samsung TV 30%`")
		return
	}

	// Parse arguments
	parts := strings.Fields(args)
	if len(parts) < 2 {
		h.sendMessage(message.Chat.ID, "❌ Você precisa especificar o produto e o preço/desconto!\n\nExemplos:\n`/add iPhone 15 R$4000`\n`/add Samsung TV 30%`")
		return
	}

	// Get the last part (price or discount)
	lastPart := parts[len(parts)-1]
	productName := strings.Join(parts[:len(parts)-1], " ")

	var targetPrice *float64
	var discountPercentage *int

	// Check if it's a percentage
	if strings.HasSuffix(lastPart, "%") {
		percentStr := strings.TrimSuffix(lastPart, "%")
		percent, err := strconv.Atoi(percentStr)
		if err != nil || percent <= 0 || percent > 100 {
			h.sendMessage(message.Chat.ID, "❌ Desconto inválido! Use um número entre 1 e 100.\n\nExemplo: `/add Samsung TV 30%`")
			return
		}
		discountPercentage = &percent
	} else {
		// Parse price
		priceStr := strings.ReplaceAll(lastPart, "R$", "")
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			h.sendMessage(message.Chat.ID, "❌ Preço inválido!\n\nExemplo: `/add iPhone 15 R$4000` ou `/add iPhone 15 4000`")
			return
		}
		targetPrice = &price
	}

	// Send add command to backend via Kafka
	h.sendCommandToBackend(models.Command{
		Type:               "add_wishlist",
		TelegramID:         message.From.ID,
		ProductName:        productName,
		TargetPrice:        targetPrice,
		DiscountPercentage: discountPercentage,
	})

	// Send confirmation
	var confirmText string
	if targetPrice != nil {
		confirmText = fmt.Sprintf("✅ *Produto adicionado!*\n\n📦 %s\n💰 Preço desejado: R$ %.2f\n\nVou te avisar quando encontrar uma oferta! 🔔",
			productName, *targetPrice)
	} else {
		confirmText = fmt.Sprintf("✅ *Produto adicionado!*\n\n📦 %s\n🔥 Desconto mínimo: %d%%\n\nVou te avisar quando encontrar uma oferta! 🔔",
			productName, *discountPercentage)
	}

	h.sendMessage(message.Chat.ID, confirmText)
}

// handleList handles the /list command
func (h *BotHandler) handleList(message *tgbotapi.Message) {
	// Send list request to backend via Kafka
	h.sendCommandToBackend(models.Command{
		Type:       "list_wishlist",
		TelegramID: message.From.ID,
		ChatID:     message.Chat.ID,
	})
	
	// Backend will respond via Kafka with the list
	h.sendMessage(message.Chat.ID, "🔍 Buscando sua lista...")
}

// handleDelete handles the /delete command
func (h *BotHandler) handleDelete(message *tgbotapi.Message) {
	args := message.CommandArguments()
	if args == "" {
		h.sendMessage(message.Chat.ID, "❌ Você precisa especificar o ID do produto!\n\nUse `/list` para ver os IDs.\n\nExemplo: `/delete 1`")
		return
	}

	id, err := strconv.Atoi(args)
	if err != nil {
		h.sendMessage(message.Chat.ID, "❌ ID inválido! Use um número.\n\nExemplo: `/delete 1`")
		return
	}

	// Send delete command to backend via Kafka
	h.sendCommandToBackend(models.Command{
		Type:       "delete_wishlist",
		TelegramID: message.From.ID,
		WishlistID: id,
		ChatID:     message.Chat.ID,
	})

	h.sendMessage(message.Chat.ID, "🗑️ Removendo produto...")
}

// SendNotification sends a notification to a user
func (h *BotHandler) SendNotification(notification *models.OfferNotification) error {
	var msg strings.Builder

	msg.WriteString("🎉 *Oferta Encontrada!*\n\n")
	msg.WriteString(fmt.Sprintf("📦 *Produto:* %s\n", notification.ProductName))
	
	if notification.Price > 0 {
		msg.WriteString(fmt.Sprintf("💰 *Preço:* R$ %.2f\n", notification.Price))
	}
	
	if notification.OriginalPrice > 0 && notification.OriginalPrice > notification.Price {
		msg.WriteString(fmt.Sprintf("~~R$ %.2f~~\n", notification.OriginalPrice))
	}
	
	if notification.DiscountPercentage > 0 {
		msg.WriteString(fmt.Sprintf("🔥 *Desconto:* %d%%\n", notification.DiscountPercentage))
	}
	
	if notification.CashbackPercentage > 0 {
		msg.WriteString(fmt.Sprintf("💸 *Cashback:* %d%%\n", notification.CashbackPercentage))
	}

	if notification.MatchType == "price" {
		msg.WriteString("\n✅ *Atingiu seu preço desejado!*")
	} else if notification.MatchType == "discount" {
		msg.WriteString("\n✅ *Atingiu o desconto desejado!*")
	}

	return h.sendMessage(notification.TelegramID, msg.String())
}

// SendWishlistResponse sends wishlist data back to user
func (h *BotHandler) SendWishlistResponse(response *models.WishlistResponse) error {
	if len(response.Items) == 0 {
		return h.sendMessage(response.ChatID, "📭 Sua lista está vazia!\n\nUse `/add` para adicionar produtos.\n\nExemplo: `/add iPhone 15 R$4000`")
	}

	var text strings.Builder
	text.WriteString("📋 *Sua Lista de Desejos*\n\n")

	for i, w := range response.Items {
		text.WriteString(fmt.Sprintf("*%d.* %s\n", i+1, w.ProductName))
		if w.TargetPrice != nil {
			text.WriteString(fmt.Sprintf("   💰 Preço: R$ %.2f\n", *w.TargetPrice))
		}
		if w.DiscountPercentage != nil {
			text.WriteString(fmt.Sprintf("   🔥 Desconto: %d%%\n", *w.DiscountPercentage))
		}
		text.WriteString(fmt.Sprintf("   🆔 ID: `%d`\n\n", w.ID))
	}

	text.WriteString(fmt.Sprintf("Total: %d produto(s)\n\n", len(response.Items)))
	text.WriteString("Para remover: `/delete <id>`")

	return h.sendMessage(response.ChatID, text.String())
}

// SendDeleteResponse sends delete confirmation
func (h *BotHandler) SendDeleteResponse(response *models.DeleteResponse) error {
	if response.Success {
		return h.sendMessage(response.ChatID, "✅ Produto removido da lista!")
	}
	return h.sendMessage(response.ChatID, "❌ Produto não encontrado!\n\nUse `/list` para ver os IDs disponíveis.")
}

// sendCommandToBackend sends a command to the backend via Kafka
func (h *BotHandler) sendCommandToBackend(cmd models.Command) error {
	cmd.Timestamp = time.Now()
	
	data, err := json.Marshal(cmd)
	if err != nil {
		log.Printf("Error marshaling command: %v", err)
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: h.commandTopic,
		Key:   sarama.StringEncoder(fmt.Sprintf("%d", cmd.TelegramID)),
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = h.kafkaProducer.SendMessage(msg)
	if err != nil {
		log.Printf("Error sending command to backend: %v", err)
		return err
	}

	return nil
}

// sendMessage sends a message to a chat
func (h *BotHandler) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}

	return nil
}
