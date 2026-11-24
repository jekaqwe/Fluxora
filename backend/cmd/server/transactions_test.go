package main

import "testing"

func TestValidateTransactionPayload(t *testing.T) {
	tests := []struct {
		name      string
		input     transactionPayload
		wantError bool
		wantTransfer bool
	}{
		{
			name: "income",
			input: transactionPayload{Title:"Зарплата", Type:"income", Amount:"1000.50", Category:"Доход", Date:"2026-11-24", AccountID:1},
		},
		{
			name: "expense",
			input: transactionPayload{Title:"Продукты", Type:"expense", Amount:"0.01", Category:"Покупки", Date:"2026-11-24", AccountID:1},
		},
		{
			name: "transfer",
			input: transactionPayload{Title:"Между счетами", Type:"transfer", Amount:"250.00", Category:"Перевод", Date:"2026-11-24", AccountID:1, TransferAccountID:2},
			wantTransfer: true,
		},
		{
			name: "zero amount",
			input: transactionPayload{Title:"Ноль", Type:"expense", Amount:"0.00", Category:"Покупки", Date:"2026-11-24", AccountID:1},
			wantError: true,
		},
		{
			name: "negative amount",
			input: transactionPayload{Title:"Минус", Type:"expense", Amount:"-1.00", Category:"Покупки", Date:"2026-11-24", AccountID:1},
			wantError: true,
		},
		{
			name: "same transfer account",
			input: transactionPayload{Title:"Самому себе", Type:"transfer", Amount:"10", Category:"Перевод", Date:"2026-11-24", AccountID:1, TransferAccountID:1},
			wantError: true,
		},
		{
			name: "invalid date",
			input: transactionPayload{Title:"Операция", Type:"income", Amount:"10", Category:"Доход", Date:"24.11.2026", AccountID:1},
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, transferID, message := validateTransactionPayload(&test.input)
			if (message != "") != test.wantError {
				t.Fatalf("validation error = %q, wantError %v", message, test.wantError)
			}
			if !test.wantError && (transferID != nil) != test.wantTransfer {
				t.Fatalf("transfer ID presence = %v, want %v", transferID != nil, test.wantTransfer)
			}
		})
	}
}
