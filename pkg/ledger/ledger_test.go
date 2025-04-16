package ledger

import (
	"testing"
	"time"
)

func TestTransactionPrint(t *testing.T) {
	tests := []struct {
		name string
		tx   Transaction
		want string
	}{
		{
			name: "basic transaction",
			tx: Transaction{
				Date:      time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC),
				Payee:     "Grocery Store",
				Account:   "expenses:food",
				Amount:    "$45.67",
				IsReceive: true,
			},
			want: `
2023/05/15 Grocery Store
    expenses:food  $45.67
    e.FIXME
`,
		},
		{
			name: "transaction with ID",
			tx: Transaction{
				Date:      time.Date(2023, 5, 16, 0, 0, 0, 0, time.UTC),
				Payee:     "Salary",
				Account:   "income:salary",
				Amount:    "$1000.00",
				IsReceive: false,
				ID:        "tx123",
			},
			want: `
2023/05/16 Salary
    income:salary  -$1000.00
    ; id: tx123
    e.FIXME
`,
		},
		{
			name: "transaction with comma",
			tx: Transaction{
				Date:      time.Date(2023, 5, 16, 0, 0, 0, 0, time.UTC),
				Payee:     "Salary",
				Account:   "income:salary",
				Amount:    "$1,000.00",
				IsReceive: false,
				ID:        "tx123",
			},
			want: `
2023/05/16 Salary
    income:salary  -$1,000.00
    ; id: tx123
    e.FIXME
`,
		},
		{
			name: "asset transaction with virtual undo",
			tx: Transaction{
				Date:      time.Date(2023, 5, 17, 0, 0, 0, 0, time.UTC),
				Payee:     "ATM Withdrawal",
				Account:   "assets:checking",
				Amount:    "$100.00",
				IsReceive: false,
			},
			want: `
2023/05/17 ATM Withdrawal
    assets:checking  -$100.00
    (assets:checking)  $100.00
    e.FIXME
`,
		},
		{
			name: "transaction with note",
			tx: Transaction{
				Date:      time.Date(2023, 5, 18, 0, 0, 0, 0, time.UTC),
				Payee:     "Coffee Shop",
				Account:   "expenses:dining",
				Amount:    "$4.50",
				IsReceive: true,
				Note:      "Business meeting",
			},
			want: `
2023/05/18 Coffee Shop
    expenses:dining  $4.50
    ; Business meeting
    e.FIXME
`,
		},
		{
			name: "comprehensive transaction",
			tx: Transaction{
				Date:      time.Date(2023, 5, 19, 0, 0, 0, 0, time.UTC),
				Payee:     "Refund",
				Account:   "assets:savings",
				Amount:    "$250.00",
				IsReceive: true,
				ID:        "refund123",
				Note:      "Store credit refund",
			},
			want: `
2023/05/19 Refund
    assets:savings  $250.00
    ; id: refund123
    (assets:savings)  -$250.00
    ; Store credit refund
    e.FIXME
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tx.Print()
			if got != tt.want {
				t.Errorf("Transaction.Print() = %v, want %v", got, tt.want)
				// Print a more readable diff to help debug
				t.Errorf("Got:\n%s\nWant:\n%s", got, tt.want)
			}
		})
	}
}
