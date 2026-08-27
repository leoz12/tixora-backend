package services

import (
	"fmt"
	"strings"

	"tixora/internal/models"
	"tixora/internal/utils"
)

// Ticket layout constants, in points. The page is sized to the ticket card
// itself rather than a full A4 sheet, so the export reads as a
// purpose-built e-ticket instead of a printed letter.
const (
	ticketPageWidth    = 600.0
	ticketMarginSide   = 20.0
	ticketMarginTop    = 20.0
	ticketMarginBottom = 44.0
	ticketCardW        = ticketPageWidth - 2*ticketMarginSide
	ticketCardH        = 380.0
	ticketPageHeight   = ticketMarginTop + ticketCardH + ticketMarginBottom
	ticketHeaderH      = 100.0
	ticketStubH        = 92.0
)

// Brand colors, matching the frontend's design tokens (src/app/globals.css)
// so the exported ticket looks like it belongs to the same product.
var (
	ticketColorBlue900  = utils.RGB{0.047, 0.173, 0.380} // #0C2C61
	ticketColorBlue700  = utils.RGB{0.078, 0.314, 0.675} // #1450AC
	ticketColorGreen700 = utils.RGB{0.039, 0.486, 0.290} // #0A7C4A
	ticketColorGreen600 = utils.RGB{0.071, 0.631, 0.314} // #12A150
	ticketColorInk900   = utils.RGB{0.059, 0.106, 0.200} // #0F1B33
	ticketColorInk500   = utils.RGB{0.392, 0.455, 0.545} // #64748B
	ticketColorLineSoft = utils.RGB{0.929, 0.945, 0.969} // #EDF1F7
	ticketColorSurface  = utils.RGB{0.961, 0.973, 0.992} // #F5F8FD
	ticketColorStubTint = utils.RGB{0.973, 0.980, 0.996}
	ticketColorShadow   = utils.RGB{0.850, 0.880, 0.930}
	ticketColorOnBlue   = utils.RGB{0.663, 0.769, 0.937} // #A9C4EF
	ticketColorWhite    = utils.RGB{1, 1, 1}
)

// buildTicketPDF renders a paid order into a short, self-contained,
// branded PDF e-ticket: a header band, event/order details, and a
// perforated stub carrying the ticket reference and amount paid.
func buildTicketPDF(order *models.Order) []byte {
	c := utils.NewPDFCanvas()

	cardX, cardY := ticketMarginSide, ticketMarginBottom
	cardTop := cardY + ticketCardH
	headerY := cardTop - ticketHeaderH
	stubTopY := cardY + ticketStubH

	// Page background — visible through the perforation notches.
	c.SetFillColor(ticketColorSurface)
	c.FillRect(0, 0, ticketPageWidth, ticketPageHeight)

	// Card drop-shadow, then the white card body on top of it.
	c.SetFillColor(ticketColorShadow)
	c.FillRect(cardX+3, cardY-3, ticketCardW, ticketCardH)
	c.SetFillColor(ticketColorWhite)
	c.FillRect(cardX, cardY, ticketCardW, ticketCardH)

	// Header band (brand gradient).
	c.GradientRect(cardX, headerY, ticketCardW, ticketHeaderH)
	c.Text(utils.FontBold, 24, ticketColorWhite, cardX+28, cardTop-42, "TIXORA")
	c.TextSpaced(utils.FontRegular, 9.5, 2, ticketColorOnBlue, cardX+28, cardTop-58, "E-TICKET")

	// "PAID" status pill, top-right of the header.
	pillW, pillH := 74.0, 26.0
	pillX := cardX + ticketCardW - 28 - pillW
	pillY := headerY + (ticketHeaderH-pillH)/2
	c.SetFillColor(ticketColorGreen600)
	c.FillRect(pillX, pillY, pillW, pillH)
	c.TextSpaced(utils.FontBold, 10, 1, ticketColorWhite, pillX+15, pillY+9, "PAID")

	// Event title + date/location.
	contentTop := headerY
	c.Text(utils.FontBold, 16, ticketColorInk900, cardX+28, contentTop-32, truncateASCII(order.Event.Title, 52))

	dateLoc := ""
	if !order.Event.EventDate.IsZero() {
		dateLoc = order.Event.EventDate.Format("Monday, 2 January 2006 - 15:04")
	}
	if order.Event.Location != "" {
		if dateLoc != "" {
			dateLoc += "  ·  " + order.Event.Location // middle dot separator
		} else {
			dateLoc = order.Event.Location
		}
	}
	if dateLoc != "" {
		c.Text(utils.FontRegular, 10.5, ticketColorInk500, cardX+28, contentTop-52, truncateASCII(dateLoc, 72))
	}

	// Divider, then a label/value info grid.
	dividerY := contentTop - 68
	c.SetStrokeColor(ticketColorLineSoft)
	c.SetLineWidth(1)
	c.Line(cardX+28, dividerY, cardX+ticketCardW-28, dividerY)

	col1X := cardX + 28
	col2X := cardX + 28 + (ticketCardW-56)/2
	row1Y := dividerY - 24
	row2Y := row1Y - 34

	drawTicketField(c, col1X, row1Y, "ORDER ID", order.OrderID)
	drawTicketField(c, col2X, row1Y, "QUANTITY", fmt.Sprintf("%d ticket(s)", order.Quantity))
	drawTicketField(c, col1X, row2Y, "BUYER", truncateASCII(order.User.Name, 30))
	drawTicketField(c, col2X, row2Y, "EMAIL", truncateASCII(order.User.Email, 34))

	// Perforated divider between the card body and the ticket stub.
	c.SetFillColor(ticketColorStubTint)
	c.FillRect(cardX, cardY, ticketCardW, ticketStubH)
	c.SetStrokeColor(utils.RGB{0.784, 0.816, 0.867}) // a touch darker than line-soft, for visible perforation
	c.SetLineWidth(1.25)
	c.SetDash(4, 4)
	c.Line(cardX, stubTopY, cardX+ticketCardW, stubTopY)
	c.SetDash(0, 0)
	c.SetFillColor(ticketColorSurface)
	c.FillCircle(cardX, stubTopY, 9)
	c.FillCircle(cardX+ticketCardW, stubTopY, 9)

	// Stub contents: reference code on the left, amount paid on the right.
	ref := "-"
	if order.TicketReference != nil {
		ref = *order.TicketReference
	}
	c.TextSpaced(utils.FontRegular, 8, 1.5, ticketColorInk500, col1X, cardY+62, "TICKET REFERENCE")
	c.TextSpaced(utils.FontBold, 19, 2, ticketColorInk900, col1X, cardY+34, ref)

	c.TextSpaced(utils.FontRegular, 8, 1.5, ticketColorInk500, col2X, cardY+62, "TOTAL PAID")
	c.Text(utils.FontBold, 17, ticketColorGreen700, col2X, cardY+32, "Rp"+formatRupiah(order.TotalPrice))

	// Footer note, below the card.
	c.Text(utils.FontItalic, 8.5, ticketColorInk500, cardX, ticketMarginBottom/2-4,
		"Show this ticket, printed or on your phone, at the venue entrance.")

	grad := utils.GradientSpec{
		X0: cardX + ticketCardW/2, Y0: headerY,
		X1: cardX + ticketCardW/2, Y1: cardTop,
		C0: ticketColorBlue700, C1: ticketColorBlue900,
	}

	return utils.RenderPDF(c, ticketPageWidth, ticketPageHeight, grad)
}

// drawTicketField renders a small uppercase label with its value beneath
// it, the info-grid building block of the ticket card.
func drawTicketField(c *utils.PDFCanvas, x, y float64, label, value string) {
	c.TextSpaced(utils.FontRegular, 7.5, 1.2, ticketColorInk500, x, y, label)
	c.Text(utils.FontBold, 11, ticketColorInk900, x, y-15, value)
}

// truncateASCII shortens s to at most max runes, appending "..." if it had
// to cut, without introducing any character outside the standard PDF
// fonts' single-byte encoding.
func truncateASCII(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

// formatRupiah renders an integer amount with "." as the thousands
// separator, matching the id-ID formatting used on the frontend.
func formatRupiah(amount int64) string {
	digits := fmt.Sprintf("%d", amount)

	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	return b.String()
}
