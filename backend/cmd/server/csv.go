package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	csvTinkoffTypeMapping = "__tinkoff_type_from_amount__"
	csvTinkoffAccountPrefix = "__tinkoff_account_id__:"
)

type importCSVRow struct {
	Date              string   `json:"date"`
	Title             string   `json:"title"`
	Type              string   `json:"type"`
	Amount            string   `json:"amount"`
	Category          string   `json:"category"`
	AccountID         int64    `json:"accountId"`
	TransferAccountID int64    `json:"transferAccountId"`
	Comment           string   `json:"comment"`
	Errors            []string `json:"errors"`
}

func csvHeaderKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", ".", "")
	return replacer.Replace(value)
}

func isTinkoffCSV(headers []string) bool {
	found := make(map[string]bool, len(headers))
	for _, header := range headers {
		found[csvHeaderKey(header)] = true
	}
	return found["датаоперации"] &&
		found["суммаоперации"] &&
		found["валютаоперации"] &&
		found["суммаввалютесчета"] &&
		found["категорияпоумолчанию"] &&
		found["статус"] &&
		found["описание"]
}

func csvFormat(headers []string) string {
	if isTinkoffCSV(headers) {
		return "tinkoff"
	}
	return "generic"
}

func csvSuggestedMapping(headers []string) map[string]string {
	if isTinkoffCSV(headers) {
		return map[string]string{
			"date": "Дата операции",
			"title": "Описание",
			"type": csvTinkoffTypeMapping,
			"amount": "Сумма операции",
			"category": "Категория по-умолчанию",
			"comment": "Сообщение",
		}
	}

	aliases := map[string][]string{
		"date":              {"date", "operationdate", "дата", "датаоперации"},
		"title":             {"title", "name", "description", "название", "наименование", "описание"},
		"type":              {"type", "kind", "operationtype", "тип", "видоперации"},
		"amount":            {"amount", "sum", "value", "сумма", "значение"},
		"category":          {"category", "категория"},
		"accountId":         {"accountid", "account", "счет", "счёт", "счетid", "счётid", "основнойсчет"},
		"transferAccountId": {"transferaccountid", "transferaccount", "счетперевода", "счётперевода", "счетполучения", "счётполучения"},
		"comment":           {"comment", "note", "комментарий", "примечание"},
	}
	normalized := make(map[string]string, len(headers))
	for _, header := range headers {
		normalized[csvHeaderKey(header)] = header
	}
	result := make(map[string]string)
	for field, choices := range aliases {
		for _, alias := range choices {
			if header, ok := normalized[csvHeaderKey(alias)]; ok {
				result[field] = header
				break
			}
		}
	}
	return result
}

func detectCSVDelimiter(data []byte) rune {
	line := strings.SplitN(strings.TrimPrefix(string(data), "\uFEFF"), "\n", 2)[0]
	commas, semicolons := 0, 0
	inQuotes := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				i++
				continue
			}
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				commas++
			}
		case ';':
			if !inQuotes {
				semicolons++
			}
		}
	}
	if semicolons > commas {
		return ';'
	}
	return ','
}

func parseCSVHeaders(data []byte) ([]string, error) {
	data = []byte(strings.TrimPrefix(string(data), "\uFEFF"))
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comma = detectCSVDelimiter(data)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}
	return header, nil
}

func parseImportCSV(data []byte) ([]importCSVRow, error) {
	return parseImportCSVWithMapping(data, nil)
}

func parseImportCSVWithMapping(data []byte, mapping map[string]string) ([]importCSVRow, error) {
	rows, _, err := parseImportCSVWithMappingStats(data, mapping)
	return rows, err
}

func parseImportCSVWithMappingStats(data []byte, mapping map[string]string) ([]importCSVRow, int, error) {
	headers, err := parseCSVHeaders(data)
	if err != nil {
		return nil, 0, err
	}
	tinkoff := isTinkoffCSV(headers)
	suggested := csvSuggestedMapping(headers)
	if mapping == nil {
		mapping = suggested
	}

	indexes := make(map[string]int, len(headers))
	for i, header := range headers {
		indexes[header] = i
	}

	fields := []string{"date", "title", "type", "amount", "category", "accountId", "transferAccountId", "comment"}
	required := map[string]bool{"date": true, "title": true, "type": true, "amount": true, "category": true, "accountId": true}
	resolved := make(map[string]int, len(fields))
	special := make(map[string]string)
	for _, field := range fields {
		source := strings.TrimSpace(mapping[field])
		if source == "" {
			if required[field] {
				return nil, 0, fmt.Errorf("Не выбрана колонка для поля %s", field)
			}
			continue
		}
		if field == "type" && tinkoff && source == csvTinkoffTypeMapping {
			special[field] = source
			continue
		}
		if field == "accountId" && tinkoff && strings.HasPrefix(source, csvTinkoffAccountPrefix) {
			special[field] = source
			continue
		}
		index, ok := indexes[source]
		if !ok {
			return nil, 0, fmt.Errorf("Колонка %q для поля %s не найдена", source, field)
		}
		resolved[field] = index
	}

	if tinkoff {
		if special["type"] != csvTinkoffTypeMapping {
			return nil, 0, fmt.Errorf("Для Tinkoff тип операции должен определяться по знаку суммы")
		}
		if special["accountId"] == "" {
			return nil, 0, fmt.Errorf("Выберите счёт Fluxora для импорта Tinkoff")
		}
	}

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\uFEFF")))
	reader.Comma = detectCSVDelimiter(data)
	reader.FieldsPerRecord = -1
	if _, err := reader.Read(); err != nil {
		return nil, 0, err
	}

	statusIndex := -1
	if tinkoff {
		for i, header := range headers {
			if csvHeaderKey(header) == "статус" {
				statusIndex = i
				break
			}
		}
	}
	rows := make([]importCSVRow, 0)
	skipped := 0
	for line := 2; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, skipped, fmt.Errorf("Строка %d: %v", line, err)
		}
		if line > 5001 {
			return nil, skipped, fmt.Errorf("Файл содержит больше 5000 строк")
		}
		if tinkoff && statusIndex >= 0 && statusIndex < len(record) && strings.TrimSpace(record[statusIndex]) != "Ок" {
			skipped++
			continue
		}

		value := func(key string) string {
			idx, ok := resolved[key]
			if !ok || idx >= len(record) {
				return ""
			}
			v := strings.TrimSpace(record[idx])
			if len(v) > 1 && v[0] == 39 && strings.ContainsRune("=+-@", rune(v[1])) {
				v = v[1:]
			}
			return strings.ReplaceAll(v, "\u00a0", " ")
		}

		amountRaw := value("amount")
		amount := normalizeImportAmount(amountRaw, tinkoff)
		kind := value("type")
		if tinkoff {
			if strings.HasPrefix(strings.TrimSpace(amountRaw), "-") {
				kind = "expense"
			} else {
				kind = "income"
			}
		}

		accountID := int64(0)
		if source := special["accountId"]; source != "" {
			accountID, _ = strconv.ParseInt(strings.TrimPrefix(source, csvTinkoffAccountPrefix), 10, 64)
		} else {
			accountID, _ = strconv.ParseInt(value("accountId"), 10, 64)
		}
		transferID, _ := strconv.ParseInt(value("transferAccountId"), 10, 64)

		if len(rows) >= 5000 {
			return nil, skipped, fmt.Errorf("Файл содержит больше 5000 строк")
		}
		rows = append(rows, importCSVRow{
			Date:              value("date"),
			Title:             value("title"),
			Type:              kind,
			Amount:            amount,
			Category:          value("category"),
			AccountID:         accountID,
			TransferAccountID: transferID,
			Comment:           value("comment"),
			Errors:            []string{},
		})
	}
	return rows, skipped, nil
}

func normalizeImportAmount(value string, absolute bool) string {
	value = strings.NewReplacer(" ", "", "\u00a0", "", "\u202f", "", ",", ".").Replace(strings.TrimSpace(value))
	if absolute {
		value = strings.TrimPrefix(value, "-")
		value = strings.TrimPrefix(value, "+")
	}
	return value
}

func parseImportDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		"2006-01-02",
		"02.01.2006 15:04:05",
		"02.01.2006",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("неподдерживаемый формат даты")
}

func validateImportRow(row *importCSVRow) {
	if _, err := parseImportDate(row.Date); err != nil {
		row.Errors = append(row.Errors, "Некорректная дата")
	}
	if row.Title == "" || len([]rune(row.Title)) > 160 {
		row.Errors = append(row.Errors, "Некорректное название")
	}
	if row.Type != "income" && row.Type != "expense" && row.Type != "transfer" {
		row.Errors = append(row.Errors, "Неизвестный тип")
	}
	if !accountAmountPattern.MatchString(row.Amount) || strings.HasPrefix(row.Amount, "-") || strings.Trim(row.Amount, "0.") == "" {
		row.Errors = append(row.Errors, "Некорректная сумма")
	}
	if row.Category == "" || len([]rune(row.Category)) > 80 {
		row.Errors = append(row.Errors, "Некорректная категория")
	}
	if row.AccountID <= 0 {
		row.Errors = append(row.Errors, "Не указан счёт")
	}
	if row.Type == "transfer" && (row.TransferAccountID <= 0 || row.TransferAccountID == row.AccountID) {
		row.Errors = append(row.Errors, "Некорректный счёт перевода")
	}
	if row.Type != "transfer" && row.TransferAccountID != 0 {
		row.Errors = append(row.Errors, "Второй счёт допустим только для перевода")
	}
	if len([]rune(row.Comment)) > 1000 {
		row.Errors = append(row.Errors, "Слишком длинный комментарий")
	}
}

func csvHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, 401)
		return
	}
	switch r.URL.Path {
	case "/api/export.csv":
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, `{"error":"Метод не поддерживается"}`, 405)
			return
		}
		fromValue, errFrom := parseOptionalDate(r.URL.Query().Get("from"))
		toValue, errTo := parseOptionalDate(r.URL.Query().Get("to"))
		if errFrom != nil || errTo != nil {
			http.Error(w, `{"error":"Некорректный период экспорта"}`, 400)
			return
		}
		if fromValue != nil && toValue != nil && fromValue.(time.Time).After(toValue.(time.Time)) {
			http.Error(w, `{"error":"Начало периода позже его окончания"}`, 400)
			return
		}
		accountFilter := int64(0)
		if raw := r.URL.Query().Get("accountId"); raw != "" {
			parsed, parseErr := strconv.ParseInt(raw, 10, 64)
			if parseErr != nil || parsed <= 0 {
				http.Error(w, `{"error":"Некорректный счёт экспорта"}`, 400)
				return
			}
			accountFilter = parsed
		}
		rows, err := database.Query(r.Context(), `SELECT operation_date::text,title,operation_type,amount::text,category,account_id,COALESCE(transfer_account_id,0),comment FROM transactions WHERE user_id=$1 AND ($2::date IS NULL OR operation_date >= $2::date) AND ($3::date IS NULL OR operation_date <= $3::date) AND ($4::bigint=0 OR account_id=$4 OR transfer_account_id=$4) AND ($5='' OR category=$5) ORDER BY operation_date,id`, userID, fromValue, toValue, accountFilter, r.URL.Query().Get("category"))
		if err != nil {
			http.Error(w, `Не удалось подготовить экспорт`, 500)
			return
		}
		defer rows.Close()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="fluxora-transactions.csv"`)
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"date", "title", "type", "amount", "category", "accountId", "transferAccountId", "comment"})
		for rows.Next() {
			var record []string
			var date, title, kind, amount, category, comment string
			var account, transfer int64
			if err := rows.Scan(&date, &title, &kind, &amount, &category, &account, &transfer, &comment); err != nil {
				return
			}
			record = []string{date, safeCSVCell(title), kind, amount, safeCSVCell(category), strconv.FormatInt(account, 10), strconv.FormatInt(transfer, 10), safeCSVCell(comment)}
			_ = writer.Write(record)
		}
		writer.Flush()
	case "/api/import/headers", "/api/import/preview", "/api/import/confirm":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, `{"error":"Метод не поддерживается"}`, 405)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			http.Error(w, `{"error":"Файл слишком большой или некорректный"}`, 400)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"Выберите CSV-файл"}`, 400)
			return
		}
		defer file.Close()
		if header.Size > 5<<20 {
			http.Error(w, `{"error":"Максимальный размер файла 5 МБ"}`, 400)
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, (5<<20)+1))
		if err != nil || len(data) > 5<<20 {
			http.Error(w, `{"error":"Не удалось прочитать CSV-файл"}`, 400)
			return
		}
		headers, headerErr := parseCSVHeaders(data)
		if headerErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка CSV: " + headerErr.Error()})
			return
		}
		if r.URL.Path == "/api/import/headers" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"headers": headers, "suggestedMapping": csvSuggestedMapping(headers), "format": csvFormat(headers),
			})
			return
		}
		mapping := map[string]string(nil)
		if raw := strings.TrimSpace(r.FormValue("mapping")); raw != "" {
			if err := json.Unmarshal([]byte(raw), &mapping); err != nil {
				http.Error(w, `{"error":"Некорректная карта колонок"}`, 400)
				return
			}
		}
		items, skipped, err := parseImportCSVWithMappingStats(data, mapping)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка CSV: " + err.Error()})
			return
		}
		valid := 0
		seenRows := make(map[string]struct{}, len(items))
		for i := range items {
			row := &items[i]
			validateImportRow(row)
			if len(row.Errors) == 0 {
				var source, destination bool
				_ = database.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`, row.AccountID, userID).Scan(&source)
				if !source {
					row.Errors = append(row.Errors, "Счёт не найден")
				}
				if row.Type == "transfer" {
					_ = database.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`, row.TransferAccountID, userID).Scan(&destination)
					if !destination {
						row.Errors = append(row.Errors, "Счёт получения не найден")
					}
				}
			}
			if len(row.Errors) == 0 {
				fingerprint := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%s", row.Date, strings.ToLower(row.Title), row.Type, normalizeCSVAmount(row.Amount), strings.ToLower(row.Category), row.AccountID, row.TransferAccountID, row.Comment)
				if _, ok := seenRows[fingerprint]; ok {
					row.Errors = append(row.Errors, "Повтор этой строки в CSV-файле")
				} else {
					seenRows[fingerprint] = struct{}{}
				}
			}
			if len(row.Errors) == 0 {
				date, _ := parseImportDate(row.Date)
				var duplicate bool
				err := database.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM transactions WHERE user_id=$1 AND operation_date=$2 AND LOWER(title)=LOWER($3) AND operation_type=$4 AND amount=$5::numeric AND LOWER(category)=LOWER($6) AND account_id=$7 AND COALESCE(transfer_account_id,0)=$8 AND comment=$9)`, userID, date, row.Title, row.Type, row.Amount, row.Category, row.AccountID, row.TransferAccountID, row.Comment).Scan(&duplicate)
				if err != nil {
					row.Errors = append(row.Errors, "Не удалось проверить дубликаты")
				} else if duplicate {
					row.Errors = append(row.Errors, "Такая операция уже существует")
				}
			}
			if len(row.Errors) == 0 {
				valid++
			}
		}
		if r.URL.Path == "/api/import/preview" {
			_ = json.NewEncoder(w).Encode(map[string]any{"rows": items, "valid": valid, "invalid": len(items) - valid, "skipped": skipped})
			return
		}
		if valid != len(items) || len(items) == 0 {
			http.Error(w, `{"error":"Импорт отменён: исправьте ошибки в предварительном просмотре"}`, 400)
			return
		}
		sum := sha256.Sum256(data)
		fileKey := hex.EncodeToString(sum[:])
		tx, err := database.Begin(r.Context())
		if err != nil {
			http.Error(w, `{"error":"Не удалось начать импорт"}`, 500)
			return
		}
		defer tx.Rollback(r.Context())
		imported := 0
		for i, item := range items {
			date, _ := parseImportDate(item.Date)
			key := "csv:" + fileKey[:40] + ":" + strconv.Itoa(i)
			tag, err := tx.Exec(r.Context(), `INSERT INTO transactions(user_id,account_id,transfer_account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,NULLIF($3,0),$4,$5,$6::numeric,$7,$8,$9,$10) ON CONFLICT(user_id,idempotency_key) DO NOTHING`, userID, item.AccountID, item.TransferAccountID, item.Title, item.Type, item.Amount, item.Category, date, item.Comment, key)
			if err != nil {
				http.Error(w, `{"error":"Не удалось импортировать строки"}`, 500)
				return
			}
			imported += int(tag.RowsAffected())
		}
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, `{"error":"Не удалось завершить импорт"}`, 500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"imported": imported, "duplicates": len(items) - imported, "skipped": skipped})
	default:
		http.NotFound(w, r)
	}
}

func parseIntDefault(v string) int64 {
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func parseOptionalDate(v string) (any, error) {
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func safeCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func normalizeCSVAmount(value string) string {
	parts := strings.SplitN(value, ".", 2)
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(parts) == 1 {
		return whole
	}
	fraction := strings.TrimRight(parts[1], "0")
	if fraction == "" {
		return whole
	}
	return whole + "." + fraction
}
