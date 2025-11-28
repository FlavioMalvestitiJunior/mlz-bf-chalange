package matcher

import (
	"fmt"
	"log"
	"strings"

	"github.com/flaviomalvestitijunior/bf-offers/backend/internal/models"
)

type OfferMatcher struct{}

func NewOfferMatcher() *OfferMatcher {
	return &OfferMatcher{}
}

// MatchOffer checks if an offer matches any wishlist items
func (m *OfferMatcher) MatchOffer(offer *models.Offer, wishlists []models.Wishlist) []models.OfferNotification {
	var notifications []models.OfferNotification

	for _, wishlist := range wishlists {
		// Check if product names match (case-insensitive, partial match)
		if !m.productMatches(offer.ProductName, wishlist.ProductName) {
			continue
		}

		// Check if price or discount matches
		matchType := ""
		matched := false

		// Check target price match
		if wishlist.TargetPrice != nil && offer.Price > 0 {
			if offer.Price <= *wishlist.TargetPrice {
				matchType = "price"
				matched = true
			}
		}

		// Check discount percentage match
		if wishlist.DiscountPercentage != nil && offer.DiscountPercentage > 0 {
			if offer.DiscountPercentage >= *wishlist.DiscountPercentage {
				matchType = "discount"
				matched = true
			}
		}

		if matched {
			notification := models.OfferNotification{
				TelegramID:         wishlist.TelegramID,
				ProductName:        offer.ProductName,
				Price:              offer.Price,
				OriginalPrice:      offer.OriginalPrice,
				DiscountPercentage: offer.DiscountPercentage,
				CashbackPercentage: offer.CashbackPercentage,
				WishlistID:         wishlist.ID,
				MatchType:          matchType,
			}
			notifications = append(notifications, notification)
			log.Printf("Match found: Product '%s' for user %d (match type: %s)",
				offer.ProductName, wishlist.TelegramID, matchType)
		}
	}

	return notifications
}

// productMatches checks if the offer product name matches the wishlist product name
// Uses case-insensitive partial matching
func (m *OfferMatcher) productMatches(offerProduct, wishlistProduct string) bool {
	offerLower := strings.ToLower(offerProduct)
	wishlistLower := strings.ToLower(wishlistProduct)

	// Check if either contains the other
	if strings.Contains(offerLower, wishlistLower) || strings.Contains(wishlistLower, offerLower) {
		return true
	}

	// Check for word-by-word matching (at least 50% of words must match)
	offerWords := strings.Fields(offerLower)
	wishlistWords := strings.Fields(wishlistLower)

	if len(wishlistWords) == 0 {
		return false
	}

	matchCount := 0
	for _, wishlistWord := range wishlistWords {
		for _, offerWord := range offerWords {
			if strings.Contains(offerWord, wishlistWord) || strings.Contains(wishlistWord, offerWord) {
				matchCount++
				break
			}
		}
	}

	matchPercentage := float64(matchCount) / float64(len(wishlistWords))
	return matchPercentage >= 0.5
}

// FormatNotificationMessage creates a formatted message for the notification
func (m *OfferMatcher) FormatNotificationMessage(notification *models.OfferNotification) string {
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

	return msg.String()
}
