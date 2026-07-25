package hooks

import (
	"kas/generated"
)

func RegisterTransactionHooks(ph *generated.ProxyHooks) {
	ph.OnTransactionsAfterCreateSuccess.BindFunc(func(e *generated.TransactionsEvent) error {
		familyID := e.PRecord.GetString("family_id")
		amount := e.PRecord.Amount()

		var incomeDelta, expenseDelta float64
		if e.PRecord.Type() == generated.Income2 {
			incomeDelta = amount
		} else {
			expenseDelta = amount
		}

		_, err := e.App.DB().NewQuery(`
			UPDATE family_balances
			SET balance = balance + {:delta},
			    total_income = total_income + {:incomeDelta},
			    total_expense = total_expense + {:expenseDelta}
			WHERE family_id = {:familyID}
		`).Bind(map[string]any{
			"delta":        incomeDelta - expenseDelta,
			"incomeDelta":  incomeDelta,
			"expenseDelta": expenseDelta,
			"familyID":     familyID,
		}).Execute()
		if err != nil {
			return err
		}
		return e.Next()
	})

	ph.OnTransactionsAfterUpdateSuccess.BindFunc(func(e *generated.TransactionsEvent) error {
		familyID := e.PRecord.GetString("family_id")

		var balance, totalIncome, totalExpense float64
		err := e.App.DB().NewQuery(`
			SELECT
				COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = {:familyID}
		`).Bind(map[string]any{"familyID": familyID}).Row(&balance, &totalIncome, &totalExpense)
		if err != nil {
			return err
		}

		_, err = e.App.DB().NewQuery(`
			UPDATE family_balances
			SET balance = {:balance}, total_income = {:totalIncome}, total_expense = {:totalExpense}
			WHERE family_id = {:familyID}
		`).Bind(map[string]any{
			"balance":      balance,
			"totalIncome":  totalIncome,
			"totalExpense": totalExpense,
			"familyID":     familyID,
		}).Execute()
		if err != nil {
			return err
		}
		return e.Next()
	})

	ph.OnTransactionsAfterDeleteSuccess.BindFunc(func(e *generated.TransactionsEvent) error {
		familyID := e.PRecord.GetString("family_id")
		amount := e.PRecord.Amount()

		var incomeDelta, expenseDelta float64
		if e.PRecord.Type() == generated.Income2 {
			incomeDelta = -amount
		} else {
			expenseDelta = -amount
		}

		_, err := e.App.DB().NewQuery(`
			UPDATE family_balances
			SET balance = balance + {:delta},
			    total_income = total_income + {:incomeDelta},
			    total_expense = total_expense + {:expenseDelta}
			WHERE family_id = {:familyID}
		`).Bind(map[string]any{
			"delta":        incomeDelta - expenseDelta,
			"incomeDelta":  incomeDelta,
			"expenseDelta": expenseDelta,
			"familyID":     familyID,
		}).Execute()
		if err != nil {
			return err
		}
		return e.Next()
	})
}
