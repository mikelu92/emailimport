package ledger

import (
	"fmt"
	"strings"
	"time"
)

type Transaction struct {
	ID        string
	Payee     string
	Amount    string
	Note      string
	Date      time.Time
	Account   string
	IsReceive bool
}

func (t Transaction) Print() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s ", t.Date.Format("2006/01/02"))
	fmt.Fprintf(&b, "%s\n", t.Payee)
	fmt.Fprintf(&b, "    %s  ", t.Account)
	if !t.IsReceive {
		fmt.Fprint(&b, "-")
	}
	fmt.Fprintf(&b, "%s\n", t.Amount)
	if t.ID != "" {
		fmt.Fprintf(&b, "    ; id: %s\n", t.ID)
	}

	// virtually undo to not mess with the unbudgeted stuff
	if strings.Index(t.Account, "assets:") == 0 {
		fmt.Fprintf(&b, "    (%s)  ", t.Account)
		if t.IsReceive {
			fmt.Fprint(&b, "-")
		}
		fmt.Fprintf(&b, "%s\n", t.Amount)
	}
	if t.Note != "" {
		fmt.Fprintf(&b, "    ; %s\n", t.Note)
	}
	fmt.Fprintf(&b, "    e.FIXME\n")
	return b.String()
}
