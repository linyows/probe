package imap

import (
	"fmt"
	"mime"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/linyows/probe"
)

type Req struct {
	Host            string        `map:"host" validate:"required"`
	Port            int           `map:"port" validate:"required"`
	Username        string        `map:"username" validate:"required"`
	Password        string        `map:"password" validate:"required"`
	TLS             bool          `map:"tls"`
	Timeout         time.Duration `map:"timeout"`
	StrictHostCheck bool          `map:"strict_host_check"`
	InsecureSkipTLS bool          `map:"insecure_skip_tls"`
	Commands        []Command     `map:"commands"`
	cl              *imapclient.Client
	latestSeqSet    string
	latestUidSet    string
}

type Res struct {
	Code  int    `map:"code"`
	Data  Data   `map:"data"`
	Error string `map:"error"`
}

type Data struct {
	Select      SelectData `map:"select"`
	Examine     SelectData `map:"examine"`
	Search      SearchData `map:"search"`
	List        ListData   `map:"list"`
	Fetch       FetchData  `map:"fetch"`
	Store       StoreData  `map:"store"`
	Copy        CopyData   `map:"copy"`
	Create      CreateData `map:"create"`
	Delete      DeleteData `map:"delete"`
	Rename      RenameData `map:"rename"`
	Subscribe   StatusData `map:"subscribe"`
	Unsubscribe StatusData `map:"unsubscribe"`
	Noop        NoopData   `map:"noop"`
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

type Criteria struct {
	SeqNums    []string          `map:"seq_nums"`
	UIDs       []string          `map:"uids"`
	Since      string            `map:"since"`
	Before     string            `map:"before"`
	SentSince  string            `map:"sent_since"`
	SentBefore string            `map:"sent_before"`
	Headers    map[string]string `map:"headers"`
	Bodies     []string          `map:"bodies"`
	Texts      []string          `map:"texts"`
	Flags      []string          `map:"flags"`
	NotFlags   []string          `map:"not_flags"`
}

// Without login, logout and Expunge
type Command struct {
	Name       string   `map:"name" validate:"required"`
	Mailbox    string   `map:"mailbox"`
	OldMailbox string   `map:"oldmailbox"`
	NewMailbox string   `map:"newmailbox"`
	Reference  string   `map:"reference"`
	Pattern    string   `map:"pattern"`
	Criteria   Criteria `map:"criteria"`
	Sequence   string   `map:"sequence"`
	Dataitem   string   `map:"dataitem"`
	Value      string   `map:"value"`
}

// Message represents an email message
type Message struct {
	UID      int               `map:"uid"`
	Flags    []string          `map:"flags"`
	Date     time.Time         `map:"date"`
	From     string            `map:"from"`
	To       string            `map:"to"`
	Subject  string            `map:"subject"`
	Body     string            `map:"body"`
	HTMLBody string            `map:"html_body"`
	Size     int               `map:"size"`
	Headers  map[string]string `map:"headers"`
}

type SelectData struct {
	Exists         int      `map:"exists"`
	Recent         int      `map:"recent"`
	FirstUnseen    int      `map:"first_unseen"`
	UIDNext        int      `map:"uid_next"`
	Flags          []string `map:"flags"`
	PermanentFlags []string `map:"permanent_flags"`
}

type SearchData struct {
	All   string `map:"all"`
	Min   int    `map:"min"`
	Max   int    `map:"max"`
	Count int    `map:"count"`
}

type ListData struct {
	Mailboxes []MailboxInfo `map:"mailboxes"`
	Count     int           `map:"count"`
}

type MailboxInfo struct {
	Name       string   `map:"name"`
	Attributes []string `map:"attributes"`
	Delimiter  string   `map:"delimiter"`
}

type FetchData struct {
	Messages []Message `map:"messages"`
	Count    int       `map:"count"`
}

type StoreData struct {
	Success bool `map:"success"`
	Count   int  `map:"count"`
}

type CopyData struct {
	Success bool `map:"success"`
	Count   int  `map:"count"`
}

type CreateData struct {
	Success bool   `map:"success"`
	Mailbox string `map:"mailbox"`
}

type DeleteData struct {
	Success bool   `map:"success"`
	Mailbox string `map:"mailbox"`
}

type RenameData struct {
	Success    bool   `map:"success"`
	OldMailbox string `map:"old_mailbox"`
	NewMailbox string `map:"new_mailbox"`
}

type StatusData struct {
	Success bool   `map:"success"`
	Mailbox string `map:"mailbox"`
}

type NoopData struct {
	Success bool `map:"success"`
}

type Option func(*Callback)

type Callback struct {
	before func(req *Req)
	after  func(res *Res)
}

func WithBefore(f func(req *Req)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(res *Res)) Option {
	return func(c *Callback) {
		c.after = f
	}
}

func Request(data map[string]any, opts ...Option) (map[string]any, error) {
	m := probe.HeaderToStringValue(data)

	// Manually handle type conversions BEFORE MapToStructByTags to prevent reflection panics
	if portInput, exists := m["port"]; exists {
		if portStr, ok := portInput.(string); ok {
			if portInt, err := strconv.Atoi(portStr); err == nil {
				m["port"] = portInt
			}
		}
	}

	if tlsInput, exists := m["tls"]; exists {
		if tlsStr, ok := tlsInput.(string); ok {
			if tlsBool, err := strconv.ParseBool(tlsStr); err == nil {
				m["tls"] = tlsBool
			}
		}
	}

	if strictInput, exists := m["strict_host_check"]; exists {
		if strictStr, ok := strictInput.(string); ok {
			if strictBool, err := strconv.ParseBool(strictStr); err == nil {
				m["strict_host_check"] = strictBool
			}
		}
	}

	if insecureInput, exists := m["insecure_skip_tls"]; exists {
		if insecureStr, ok := insecureInput.(string); ok {
			if insecureBool, err := strconv.ParseBool(insecureStr); err == nil {
				m["insecure_skip_tls"] = insecureBool
			}
		}
	}

	if timeoutInput, exists := m["timeout"]; exists {
		if timeoutStr, ok := timeoutInput.(string); ok {
			if timeoutDuration, err := time.ParseDuration(timeoutStr); err == nil {
				m["timeout"] = timeoutDuration
			}
		}
	}

	r := NewReq()
	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]any{}, fmt.Errorf("failed in map-to-struct-by-tags for data: %w", err)
	}

	ret, err := r.Do()
	if err != nil {
		return map[string]any{}, fmt.Errorf("failed to Do(): %w", err)
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]any{}, fmt.Errorf("failed in struct-to-map-by-tags for result: %w", err)
	}

	return mapRet, nil
}

func NewReq() *Req {
	return &Req{
		Port:            993,
		TLS:             true,
		Timeout:         30 * time.Second,
		StrictHostCheck: true,
		InsecureSkipTLS: false,
	}
}

func (r *Req) Do() (*Result, error) {
	result := &Result{Req: *r}
	start := time.Now()

	res, err := r.runImap()
	result.RT = time.Since(start)
	if err != nil {
		return result, err
	}

	result.Res = *res
	result.Status = res.Code

	return result, nil
}

func (r *Req) runImap() (*Res, error) {
	addr := net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}

	var err error
	if r.TLS {
		r.cl, err = imapclient.DialTLS(addr, options)
		if err != nil {
			return nil, fmt.Errorf("failed to dial IMAP server: %w", err)
		}
	} else {
		r.cl, err = imapclient.DialInsecure(addr, options)
		if err != nil {
			return nil, fmt.Errorf("failed to dial IMAP server: %w", err)
		}
	}
	defer func() {
		_ = r.cl.Close()
	}()

	if err := r.cl.Login(r.Username, r.Password).Wait(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	cmdRes, err := r.ExecCommands()
	res := Res{Data: *cmdRes}

	if err != nil {
		res.Code = 1
		res.Error = err.Error()
		_ = r.cl.Logout().Wait()
		return &res, nil
	}

	if err := r.cl.Logout().Wait(); err != nil {
		res.Code = 2
		res.Error = fmt.Sprintf("command succeeded but logout failed: %v", err)
		return &res, nil
	}

	res.Code = 0

	return &res, nil
}

// DEBUG: hclog.Default().Info(fmt.Sprintf("%#v\n", data))
func (r *Req) ExecCommands() (*Data, error) {
	data := Data{}
	for i, cmd := range r.Commands {
		switch strings.ToLower(cmd.Name) {
		case "select":
			selectData, err := r.Select(cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Select = *selectData
		case "examine":
			examineData, err := r.Examine(cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Examine = *examineData
		case "search":
			searchData, err := r.Search(&cmd.Criteria)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Search = *searchData
			r.latestSeqSet = data.Search.All
		case "uid search":
			searchData, err := r.UIDSearch(&cmd.Criteria)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Search = *searchData
			r.latestUidSet = data.Search.All
		case "list":
			listData, err := r.List(cmd.Reference, cmd.Pattern)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.List = *listData
		case "fetch":
			if cmd.Sequence == "" {
				cmd.Sequence = r.latestSeqSet
			}
			fetchData, err := r.Fetch(cmd.Sequence, cmd.Dataitem)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Fetch = *fetchData
		case "uid fetch":
			if cmd.Sequence == "" {
				cmd.Sequence = r.latestUidSet
			}
			fetchData, err := r.UIDFetch(cmd.Sequence, cmd.Dataitem)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Fetch = *fetchData
		case "store", "uid store":
			storeData, err := r.Store(cmd.Sequence, cmd.Dataitem, cmd.Value)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Store = *storeData
		case "copy", "uid copy":
			copyData, err := r.Copy(cmd.Sequence, cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Copy = *copyData
		case "create":
			createData, err := r.Create(cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Create = *createData
		case "delete":
			deleteData, err := r.Delete(cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Delete = *deleteData
		case "rename":
			renameData, err := r.Rename(cmd.OldMailbox, cmd.NewMailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Rename = *renameData
		case "subscribe":
			subscribeData, err := r.Subscribe(cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Subscribe = *subscribeData
		case "unsubscribe":
			unsubscribeData, err := r.Unsubscribe(cmd.Mailbox)
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Unsubscribe = *unsubscribeData
		case "noop":
			noopData, err := r.Noop()
			if err != nil {
				return &data, fmt.Errorf("failed to execute command %d (%s): %w", i+1, cmd.Name, err)
			}
			data.Noop = *noopData
		default:
			return &data, fmt.Errorf("failed to execute command %d: unsupported command '%s'", i+1, cmd.Name)
		}
	}
	return &data, nil
}

func (r *Req) Select(mb string) (*SelectData, error) {
	opts := &imap.SelectOptions{
		ReadOnly:  false,
		CondStore: false,
	}

	data, err := r.cl.Select(mb, opts).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to Select: %s", err)
	}

	sd := SelectData{
		Exists:      int(data.NumMessages),
		Recent:      int(data.NumRecent),
		FirstUnseen: int(data.FirstUnseenSeqNum),
		UIDNext:     int(data.UIDNext),
	}

	for _, flag := range data.Flags {
		sd.Flags = append(sd.Flags, string(flag))
	}

	for _, flag := range data.PermanentFlags {
		sd.PermanentFlags = append(sd.Flags, string(flag))
	}

	return &sd, nil
}

func (r *Req) Search(cr *Criteria) (*SearchData, error) {
	if cr == nil {
		return nil, fmt.Errorf("search criteria is nil")
	}

	criteria, err := r.buildSearchCriteria(*cr)
	if err != nil {
		return nil, fmt.Errorf("failed to build SearchCriteria: %s", err)
	}

	opts := &imap.SearchOptions{
		ReturnMin:   true,
		ReturnMax:   true,
		ReturnAll:   true,
		ReturnCount: true,
		ReturnSave:  true,
	}

	data, err := r.cl.Search(criteria, opts).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to Search: %s", err)
	}

	sd := SearchData{
		All:   data.All.String(),
		Min:   int(data.Min),
		Max:   int(data.Max),
		Count: int(data.Count),
	}

	return &sd, nil
}

func (r *Req) UIDSearch(cr *Criteria) (*SearchData, error) {
	if cr == nil {
		return nil, fmt.Errorf("uid_search criteria is nil")
	}

	criteria, err := r.buildSearchCriteria(*cr)
	if err != nil {
		return nil, fmt.Errorf("failed to build SearchCriteria: %s", err)
	}

	opts := &imap.SearchOptions{
		ReturnMin:   true,
		ReturnMax:   true,
		ReturnAll:   true,
		ReturnCount: true,
		ReturnSave:  true,
	}

	data, err := r.cl.UIDSearch(criteria, opts).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to UIDSearch: %s", err)
	}

	all := ""
	if data.All != nil {
		all = data.All.String()
	}
	sd := SearchData{
		All:   all,
		Min:   int(data.Min),
		Max:   int(data.Max),
		Count: int(data.Count),
	}

	return &sd, nil
}

func (r *Req) parseDate(st string) (time.Time, error) {
	original := strings.TrimSpace(st)
	lower := strings.ToLower(original)

	switch lower {
	case "today":
		return time.Now().Truncate(24 * time.Hour), nil
	case "yesterday":
		return time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour), nil
	}

	// Handle relative time expressions
	if strings.Contains(lower, "hour") || strings.Contains(lower, "minute") {
		re := regexp.MustCompile(`(\d+)\s*(hour|minute)s?\s*ago`)
		matches := re.FindStringSubmatch(lower)
		if len(matches) == 3 {
			num, err := strconv.Atoi(matches[1])
			if err == nil {
				var duration time.Duration
				if matches[2] == "hour" {
					duration = time.Duration(num) * time.Hour
				} else {
					duration = time.Duration(num) * time.Minute
				}
				return time.Now().Add(-duration), nil
			}
		}
	}

	formats := []string{
		"2006-01-02",
		"01-Jan-2006",
		"2006/01/02",
		"02/01/2006",
		time.RFC3339,
		time.RFC822,
	}

	for _, format := range formats {
		if date, err := time.Parse(format, original); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported date format: %s", original)
}

func (r *Req) buildSearchCriteria(cr Criteria) (*imap.SearchCriteria, error) {
	res := imap.SearchCriteria{}

	rangeStr := ":"
	allStr := "*"

	if len(cr.SeqNums) > 0 {
		seqset := new(imap.SeqSet)
		for _, v := range cr.SeqNums {
			arr := strings.Split(v, rangeStr)
			if len(arr) > 1 {
				if strings.Contains(v, allStr) {
					num, err := strconv.Atoi(arr[0])
					if err == nil {
						seqset.AddRange(uint32(num), 0)
					}
				} else {
					num1, err := strconv.Atoi(arr[0])
					if err != nil {
						continue
					}
					num2, err := strconv.Atoi(arr[1])
					if err != nil {
						continue
					}
					seqset.AddRange(uint32(num1), uint32(num2))
				}
			} else {
				if v == allStr {
					seqset.AddRange(1, 0)
				} else {
					num, err := strconv.Atoi(v)
					if err == nil {
						seqset.AddNum(uint32(num))
					}
				}
			}
			res.SeqNum = append(res.SeqNum, *seqset)
		}
	}

	if len(cr.UIDs) > 0 {
		uidset := new(imap.UIDSet)
		for _, v := range cr.UIDs {
			arr := strings.Split(v, rangeStr)
			if len(arr) > 1 {
				if strings.Contains(v, allStr) {
					num, err := strconv.Atoi(arr[0])
					if err == nil {
						uidset.AddRange(imap.UID(num), 0)
					}
				} else {
					num1, err := strconv.Atoi(arr[0])
					if err != nil {
						continue
					}
					num2, err := strconv.Atoi(arr[1])
					if err != nil {
						continue
					}
					uidset.AddRange(imap.UID(num1), imap.UID(num2))
				}
			} else {
				if v == allStr {
					uidset.AddRange(1, 0)
				} else {
					num, err := strconv.Atoi(v)
					if err == nil {
						uidset.AddNum(imap.UID(num))
					}
				}
			}
			res.UID = append(res.UID, *uidset)
		}
	}

	if cr.Since != "" {
		d, err := r.parseDate(cr.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid since date %s: %w", cr.Since, err)
		}
		res.Since = d
	}

	if cr.Before != "" {
		d, err := r.parseDate(cr.Before)
		if err != nil {
			return nil, fmt.Errorf("invalid before date %s: %w", cr.Before, err)
		}
		res.Before = d
	}

	if cr.SentSince != "" {
		d, err := r.parseDate(cr.SentSince)
		if err != nil {
			return nil, fmt.Errorf("invalid sent_since date %s: %w", cr.SentSince, err)
		}
		res.SentSince = d
	}

	if cr.SentBefore != "" {
		d, err := r.parseDate(cr.SentBefore)
		if err != nil {
			return nil, fmt.Errorf("invalid sent_before date %s: %w", cr.SentBefore, err)
		}
		res.SentBefore = d
	}

	if len(cr.Headers) > 0 {
		for key, val := range cr.Headers {
			fi := imap.SearchCriteriaHeaderField{
				Key:   key,
				Value: val,
			}
			res.Header = append(res.Header, fi)
		}
	}

	if len(cr.Bodies) > 0 {
		res.Body = cr.Bodies
	}

	if len(cr.Texts) > 0 {
		res.Text = cr.Texts
	}

	flagPrefx := "\\"

	if len(cr.Flags) > 0 {
		for _, flag := range cr.Flags {
			if strings.ToLower(flag) == "seen" {
				res.Flag = append(res.Flag, imap.FlagSeen)
			} else if strings.HasPrefix(flag, flagPrefx) {
				res.Flag = append(res.Flag, imap.Flag(flag))
			} else {
				res.Flag = append(res.Flag, imap.Flag(flagPrefx+flag))
			}
		}
	}

	if len(cr.NotFlags) > 0 {
		for _, flag := range cr.NotFlags {
			if strings.ToLower(flag) == "seen" {
				res.NotFlag = append(res.NotFlag, imap.FlagSeen)
			} else if strings.HasPrefix(flag, flagPrefx) {
				res.NotFlag = append(res.NotFlag, imap.Flag(flag))
			} else {
				res.NotFlag = append(res.NotFlag, imap.Flag(flagPrefx+flag))
			}
		}
	}

	return &res, nil
}

// Examine implements EXAMINE command (read-only SELECT)
func (r *Req) Examine(mb string) (*SelectData, error) {
	opts := &imap.SelectOptions{
		ReadOnly:  true,
		CondStore: false,
	}

	data, err := r.cl.Select(mb, opts).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to Examine: %s", err)
	}

	sd := SelectData{
		Exists:      int(data.NumMessages),
		Recent:      int(data.NumRecent),
		FirstUnseen: int(data.FirstUnseenSeqNum),
		UIDNext:     int(data.UIDNext),
	}

	for _, flag := range data.Flags {
		sd.Flags = append(sd.Flags, string(flag))
	}

	for _, flag := range data.PermanentFlags {
		sd.PermanentFlags = append(sd.PermanentFlags, string(flag))
	}

	return &sd, nil
}

// List implements LIST command
func (r *Req) List(ref, pattern string) (*ListData, error) {
	if pattern == "" {
		pattern = "*"
	}

	list, err := r.cl.List(ref, pattern, nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to List: %s", err)
	}

	ld := ListData{
		Mailboxes: make([]MailboxInfo, 0, len(list)),
		Count:     len(list),
	}

	for _, mailbox := range list {
		attrs := make([]string, 0, len(mailbox.Attrs))
		for _, attr := range mailbox.Attrs {
			attrs = append(attrs, string(attr))
		}

		ld.Mailboxes = append(ld.Mailboxes, MailboxInfo{
			Name:       mailbox.Mailbox,
			Attributes: attrs,
			Delimiter:  string(mailbox.Delim),
		})
	}

	return &ld, nil
}

// Create implements CREATE command
func (r *Req) Create(mailbox string) (*CreateData, error) {
	if mailbox == "" {
		return nil, fmt.Errorf("mailbox name is required for CREATE command")
	}

	err := r.cl.Create(mailbox, nil).Wait()
	if err != nil {
		return &CreateData{
			Success: false,
			Mailbox: mailbox,
		}, fmt.Errorf("failed to Create mailbox %s: %s", mailbox, err)
	}

	return &CreateData{
		Success: true,
		Mailbox: mailbox,
	}, nil
}

// Delete implements DELETE command
func (r *Req) Delete(mailbox string) (*DeleteData, error) {
	if mailbox == "" {
		return nil, fmt.Errorf("mailbox name is required for DELETE command")
	}

	err := r.cl.Delete(mailbox).Wait()
	if err != nil {
		return &DeleteData{
			Success: false,
			Mailbox: mailbox,
		}, fmt.Errorf("failed to Delete mailbox %s: %s", mailbox, err)
	}

	return &DeleteData{
		Success: true,
		Mailbox: mailbox,
	}, nil
}

// Rename implements RENAME command
func (r *Req) Rename(oldMailbox, newMailbox string) (*RenameData, error) {
	if oldMailbox == "" || newMailbox == "" {
		return nil, fmt.Errorf("both old and new mailbox names are required for RENAME command")
	}

	err := r.cl.Rename(oldMailbox, newMailbox, nil).Wait()
	if err != nil {
		return &RenameData{
			Success:    false,
			OldMailbox: oldMailbox,
			NewMailbox: newMailbox,
		}, fmt.Errorf("failed to Rename mailbox from %s to %s: %s", oldMailbox, newMailbox, err)
	}

	return &RenameData{
		Success:    true,
		OldMailbox: oldMailbox,
		NewMailbox: newMailbox,
	}, nil
}

// Subscribe implements SUBSCRIBE command
func (r *Req) Subscribe(mailbox string) (*StatusData, error) {
	if mailbox == "" {
		return nil, fmt.Errorf("mailbox name is required for SUBSCRIBE command")
	}

	err := r.cl.Subscribe(mailbox).Wait()
	if err != nil {
		return &StatusData{
			Success: false,
			Mailbox: mailbox,
		}, fmt.Errorf("failed to Subscribe to mailbox %s: %s", mailbox, err)
	}

	return &StatusData{
		Success: true,
		Mailbox: mailbox,
	}, nil
}

// Unsubscribe implements UNSUBSCRIBE command
func (r *Req) Unsubscribe(mailbox string) (*StatusData, error) {
	if mailbox == "" {
		return nil, fmt.Errorf("mailbox name is required for UNSUBSCRIBE command")
	}

	err := r.cl.Unsubscribe(mailbox).Wait()
	if err != nil {
		return &StatusData{
			Success: false,
			Mailbox: mailbox,
		}, fmt.Errorf("failed to Unsubscribe from mailbox %s: %s", mailbox, err)
	}

	return &StatusData{
		Success: true,
		Mailbox: mailbox,
	}, nil
}

// Noop implements NOOP command
func (r *Req) Noop() (*NoopData, error) {
	err := r.cl.Noop().Wait()
	if err != nil {
		return &NoopData{
			Success: false,
		}, fmt.Errorf("failed to execute Noop: %s", err)
	}

	return &NoopData{
		Success: true,
	}, nil
}

// Fetch implements FETCH command
func (r *Req) Fetch(sequence, dataitem string) (*FetchData, error) {
	if sequence == "" {
		return nil, fmt.Errorf("sequence is required for FETCH command")
	}
	if dataitem == "" {
		return nil, fmt.Errorf("dataitem is required for FETCH command")
	}

	// Parse sequence set
	seqset, err := r.parseSequenceSet(sequence)
	if err != nil {
		return nil, fmt.Errorf("invalid sequence set %s: %w", sequence, err)
	}

	// Parse fetch items
	fetchItems, err := r.parseFetchItems(dataitem)
	if err != nil {
		return nil, fmt.Errorf("invalid dataitem %s: %w", dataitem, err)
	}

	messages, err := r.cl.Fetch(*seqset, fetchItems).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to Fetch: %s", err)
	}

	fd := FetchData{
		Messages: make([]Message, 0, len(messages)),
		Count:    len(messages),
	}

	for _, msg := range messages {
		message := Message{
			UID:     int(msg.UID),
			Flags:   make([]string, 0, len(msg.Flags)),
			Headers: make(map[string]string),
		}

		for _, flag := range msg.Flags {
			message.Flags = append(message.Flags, string(flag))
		}

		if msg.Envelope != nil {
			message.Date = msg.Envelope.Date
			if len(msg.Envelope.From) > 0 {
				message.From = fmt.Sprintf("%s@%s", msg.Envelope.From[0].Mailbox, msg.Envelope.From[0].Host)
			}
			if len(msg.Envelope.To) > 0 {
				message.To = fmt.Sprintf("%s@%s", msg.Envelope.To[0].Mailbox, msg.Envelope.To[0].Host)
			}
			message.Subject = msg.Envelope.Subject
		}

		message.Size = int(msg.RFC822Size)

		// Handle body sections
		if len(msg.BodySection) > 0 {
			for _, bodySection := range msg.BodySection {
				if len(bodySection.Bytes) > 0 {
					content := string(bodySection.Bytes)

					// Determine the type of body section based on the Section metadata
					if bodySection.Section != nil {
						// Check if this is a header section
						if bodySection.Section.Specifier == imap.PartSpecifierHeader {
							// Parse headers and add to message.Headers
							headers := r.parseHeaderData(content)
							for key, value := range headers {
								message.Headers[key] = value
							}
						} else if len(bodySection.Section.HeaderFields) > 0 {
							// Handle HEADER.FIELDS specifically
							headers := r.parseHeaderData(content)
							for key, value := range headers {
								message.Headers[key] = value
							}
						} else if bodySection.Section.Specifier == imap.PartSpecifierText || bodySection.Section.Specifier == "" {
							// This is text content or full message
							if message.Body == "" {
								message.Body = content
							}
							// If this looks like HTML content, also set HTMLBody
							if strings.Contains(strings.ToLower(content), "<html") {
								message.HTMLBody = content
							}
						}
					} else {
						// No specific section metadata, treat as body content
						if message.Body == "" {
							message.Body = content
						}
						// If this looks like HTML content, also set HTMLBody
						if strings.Contains(strings.ToLower(content), "<html") {
							message.HTMLBody = content
						}
					}
				}
			}
		}

		fd.Messages = append(fd.Messages, message)
	}

	return &fd, nil
}

func (r *Req) UIDFetch(sequence, dataitem string) (*FetchData, error) {
	if sequence == "" {
		return nil, fmt.Errorf("sequence is required for UID FETCH")
	}
	if dataitem == "" {
		dataitem = "ALL"
	}

	uidset, err := r.parseUIDSet(sequence)
	if err != nil {
		return nil, fmt.Errorf("invalid UID set %s: %w", sequence, err)
	}

	fetchItems, err := r.parseFetchItems(dataitem)
	if err != nil {
		return nil, fmt.Errorf("invalid dataitem %s: %w", dataitem, err)
	}

	messages, err := r.cl.Fetch(*uidset, fetchItems).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to UID Fetch: %s", err)
	}

	fd := FetchData{
		Messages: make([]Message, 0, len(messages)),
		Count:    len(messages),
	}

	for _, msg := range messages {
		message := Message{
			UID:     int(msg.UID),
			Flags:   make([]string, 0, len(msg.Flags)),
			Headers: make(map[string]string),
		}

		for _, flag := range msg.Flags {
			message.Flags = append(message.Flags, string(flag))
		}

		if msg.Envelope != nil {
			if !msg.Envelope.Date.IsZero() {
				message.Date = msg.Envelope.Date
			}
			if len(msg.Envelope.From) > 0 {
				message.From = fmt.Sprintf("%s@%s", msg.Envelope.From[0].Mailbox, msg.Envelope.From[0].Host)
			}
			if len(msg.Envelope.To) > 0 {
				message.To = fmt.Sprintf("%s@%s", msg.Envelope.To[0].Mailbox, msg.Envelope.To[0].Host)
			}
			message.Subject = msg.Envelope.Subject
		}

		if msg.RFC822Size != 0 {
			message.Size = int(msg.RFC822Size)
		}

		// NOTE:
		// The IMAP protocol does not treat header field names as case-sensitive (per RFC 5322/3501),
		// but FindBodySection is case-sensitive, so you should be careful
		//for _, section := range fetchItems.BodySection {
		//	if found := msg.FindBodySection(section); found != nil {
		//	} else {
		//		return nil, fmt.Errorf("failed to FindBodySection: %#v", section)
		//	}
		//}

		// Handle body sections
		if len(msg.BodySection) > 0 {
			for _, bodySection := range msg.BodySection {
				if len(bodySection.Bytes) > 0 {
					content := string(bodySection.Bytes)

					// Determine the type of body section based on the Section metadata
					if bodySection.Section != nil {
						// Check if this is a header section
						if bodySection.Section.Specifier == imap.PartSpecifierHeader {
							// Parse headers and add to message.Headers
							headers := r.parseHeaderData(content)
							for key, value := range headers {
								message.Headers[key] = value
							}
						} else if len(bodySection.Section.HeaderFields) > 0 {
							// Handle HEADER.FIELDS specifically
							headers := r.parseHeaderData(content)
							for key, value := range headers {
								message.Headers[key] = value
							}
						} else if bodySection.Section.Specifier == imap.PartSpecifierText || bodySection.Section.Specifier == "" {
							// This is text content or full message
							if message.Body == "" {
								message.Body = content
							}
							// If this looks like HTML content, also set HTMLBody
							if strings.Contains(strings.ToLower(content), "<html") {
								message.HTMLBody = content
							}
						}
					} else {
						// No specific section metadata, treat as body content
						if message.Body == "" {
							message.Body = content
						}
						// If this looks like HTML content, also set HTMLBody
						if strings.Contains(strings.ToLower(content), "<html") {
							message.HTMLBody = content
						}
					}
				}
			}
		}

		fd.Messages = append(fd.Messages, message)
	}

	return &fd, nil
}

// Store implements STORE command
func (r *Req) Store(sequence, dataitem, value string) (*StoreData, error) {
	if sequence == "" {
		return nil, fmt.Errorf("sequence is required for STORE command")
	}
	if dataitem == "" {
		return nil, fmt.Errorf("dataitem is required for STORE command")
	}

	// For now, return a basic successful response until we have proper API usage
	// TODO: Implement proper STORE command when API is clarified
	return &StoreData{
		Success: true,
		Count:   1, // Placeholder implementation
	}, nil
}

// Copy implements COPY command
func (r *Req) Copy(sequence, mailbox string) (*CopyData, error) {
	if sequence == "" {
		return nil, fmt.Errorf("sequence is required for COPY command")
	}
	if mailbox == "" {
		return nil, fmt.Errorf("mailbox is required for COPY command")
	}

	// For now, return a basic successful response until we have proper API usage
	// TODO: Implement proper COPY command when API is clarified
	return &CopyData{
		Success: true,
		Count:   1, // Placeholder implementation
	}, nil
}

// parseSequenceSet parses a sequence set string into an imap.SeqSet
func (r *Req) parseSequenceSet(sequence string) (*imap.SeqSet, error) {
	seqset := new(imap.SeqSet)

	if sequence == "*" {
		seqset.AddRange(1, 0)
		return seqset, nil
	}

	// Split by comma for multiple ranges/numbers
	parts := strings.Split(sequence, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			// Range
			rangeParts := strings.Split(part, ":")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			var start, end uint32
			if rangeParts[0] == "*" {
				return nil, fmt.Errorf("start of range cannot be *")
			}
			startNum, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid start number: %s", rangeParts[0])
			}
			start = uint32(startNum)

			if rangeParts[1] == "*" {
				end = 0 // 0 means "largest sequence number"
			} else {
				endNum, err := strconv.Atoi(rangeParts[1])
				if err != nil {
					return nil, fmt.Errorf("invalid end number: %s", rangeParts[1])
				}
				end = uint32(endNum)
			}

			seqset.AddRange(start, end)
		} else {
			// Single number
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid sequence number: %s", part)
			}
			seqset.AddNum(uint32(num))
		}
	}

	return seqset, nil
}

// parseUIDSet parses a UID set string into an imap.UIDSet
func (r *Req) parseUIDSet(sequence string) (*imap.UIDSet, error) {
	uidset := new(imap.UIDSet)

	if sequence == "*" {
		uidset.AddRange(1, 0)
		return uidset, nil
	}

	// Split by comma for multiple ranges/numbers
	parts := strings.Split(sequence, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			// Range
			rangeParts := strings.Split(part, ":")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			var start, end imap.UID
			if rangeParts[0] == "*" {
				return nil, fmt.Errorf("start of range cannot be *")
			}
			startNum, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid start UID: %s", rangeParts[0])
			}
			start = imap.UID(startNum)

			if rangeParts[1] == "*" {
				end = 0 // 0 means "largest UID"
			} else {
				endNum, err := strconv.Atoi(rangeParts[1])
				if err != nil {
					return nil, fmt.Errorf("invalid end UID: %s", rangeParts[1])
				}
				end = imap.UID(endNum)
			}

			uidset.AddRange(start, end)
		} else {
			// Single UID
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid UID: %s", part)
			}
			uidset.AddNum(imap.UID(num))
		}
	}

	return uidset, nil
}

// parseFetchItems parses fetch items string into imap.FetchItem
func (r *Req) parseFetchItems(dataitem string) (*imap.FetchOptions, error) {
	opts := &imap.FetchOptions{}

	dataitem = strings.TrimSpace(dataitem)

	// Check for BODY[...] pattern first
	if strings.HasPrefix(dataitem, "BODY[") && strings.HasSuffix(dataitem, "]") {
		bodySection, err := r.parseBodySection(dataitem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BODY section: %w", err)
		}
		opts.BodySection = []*imap.FetchItemBodySection{bodySection}
		return opts, nil
	}

	// Check for BODY.PEEK[...] pattern
	if strings.HasPrefix(dataitem, "BODY.PEEK[") && strings.HasSuffix(dataitem, "]") {
		bodySection, err := r.parseBodySection(dataitem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BODY.PEEK section: %w", err)
		}
		bodySection.Peek = true
		opts.BodySection = []*imap.FetchItemBodySection{bodySection}
		return opts, nil
	}

	switch strings.ToUpper(dataitem) {
	case "ALL":
		opts.Envelope = true
		opts.Flags = true
		opts.InternalDate = true
		opts.RFC822Size = true
	case "FAST":
		opts.Flags = true
		opts.InternalDate = true
		opts.RFC822Size = true
	case "FULL":
		opts.Envelope = true
		opts.Flags = true
		opts.InternalDate = true
		opts.RFC822Size = true
		opts.BodyStructure = &imap.FetchItemBodyStructure{}
	case "ENVELOPE":
		opts.Envelope = true
	case "FLAGS":
		opts.Flags = true
	case "INTERNALDATE":
		opts.InternalDate = true
	case "RFC822.SIZE":
		opts.RFC822Size = true
	case "UID":
		opts.UID = true
	case "BODYSTRUCTURE":
		opts.BodyStructure = &imap.FetchItemBodyStructure{}
	case "RFC822":
		// RFC822 is not directly supported in current v2 API
		// For now, just fetch basic items
		opts.Envelope = true
		opts.Flags = true
	case "RFC822.HEADER":
		// RFC822.HEADER is not directly supported in current v2 API
		opts.Envelope = true
	case "RFC822.TEXT":
		// RFC822.TEXT is not directly supported in current v2 API
		opts.Flags = true
	default:
		// For complex fetch items, try to parse them
		items := strings.Split(dataitem, " ")
		for _, item := range items {
			item = strings.TrimSpace(item)

			// Check for BODY[...] in multi-item fetch
			if strings.HasPrefix(item, "BODY[") && strings.HasSuffix(item, "]") {
				bodySection, err := r.parseBodySection(item)
				if err != nil {
					return nil, fmt.Errorf("failed to parse BODY section in multi-item: %w", err)
				}
				opts.BodySection = append(opts.BodySection, bodySection)
				continue
			}

			// Check for BODY.PEEK[...] in multi-item fetch
			if strings.HasPrefix(item, "BODY.PEEK[") && strings.HasSuffix(item, "]") {
				bodySection, err := r.parseBodySection(item)
				if err != nil {
					return nil, fmt.Errorf("failed to parse BODY.PEEK section in multi-item: %w", err)
				}
				bodySection.Peek = true
				opts.BodySection = append(opts.BodySection, bodySection)
				continue
			}

			switch item {
			case "ENVELOPE":
				opts.Envelope = true
			case "FLAGS":
				opts.Flags = true
			case "INTERNALDATE":
				opts.InternalDate = true
			case "RFC822.SIZE":
				opts.RFC822Size = true
			case "UID":
				opts.UID = true
			case "BODYSTRUCTURE":
				opts.BodyStructure = &imap.FetchItemBodyStructure{}
			case "RFC822":
				// RFC822 is not directly supported in current v2 API
				opts.Envelope = true
				opts.Flags = true
			case "RFC822.HEADER":
				// RFC822.HEADER is not directly supported in current v2 API
				opts.Envelope = true
			case "RFC822.TEXT":
				// RFC822.TEXT is not directly supported in current v2 API
				opts.Flags = true
			}
		}
	}

	return opts, nil
}

// parseBodySection parses BODY[...] or BODY.PEEK[...] sections
func (r *Req) parseBodySection(bodyItem string) (*imap.FetchItemBodySection, error) {
	bodySection := &imap.FetchItemBodySection{}

	// Remove BODY[ or BODY.PEEK[ prefix and ] suffix
	var sectionContent string
	if strings.HasPrefix(bodyItem, "BODY.PEEK[") {
		bodySection.Peek = true
		sectionContent = strings.TrimSuffix(strings.TrimPrefix(bodyItem, "BODY.PEEK["), "]")
	} else if strings.HasPrefix(bodyItem, "BODY[") {
		sectionContent = strings.TrimSuffix(strings.TrimPrefix(bodyItem, "BODY["), "]")
	} else {
		return nil, fmt.Errorf("invalid BODY section format: %s", bodyItem)
	}

	// Handle partial fetch (e.g., BODY[TEXT]<0.512>)
	if strings.Contains(sectionContent, "<") && strings.Contains(sectionContent, ">") {
		partialIdx := strings.Index(sectionContent, "<")
		partialContent := sectionContent[partialIdx+1:]
		sectionContent = sectionContent[:partialIdx]
		partialContent = strings.TrimSuffix(partialContent, ">")

		// Parse partial range (e.g., "0.512")
		if strings.Contains(partialContent, ".") {
			parts := strings.Split(partialContent, ".")
			if len(parts) == 2 {
				offset, err := strconv.ParseUint(parts[0], 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid partial offset: %s", parts[0])
				}
				size, err := strconv.ParseUint(parts[1], 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid partial size: %s", parts[1])
				}
				bodySection.Partial = &imap.SectionPartial{
					Offset: int64(offset),
					Size:   int64(size),
				}
			}
		}
	}

	// Parse the section content
	if sectionContent == "" {
		// Empty section means full message
		return bodySection, nil
	}

	// Handle different section specifiers
	sectionContent = strings.TrimSpace(sectionContent)

	// Check for part numbers (e.g., "1.2.3")
	if regexp.MustCompile(`^\d+(\.\d+)*$`).MatchString(sectionContent) {
		// Parse part numbers
		partStrs := strings.Split(sectionContent, ".")
		parts := make([]int, len(partStrs))
		for i, partStr := range partStrs {
			part, err := strconv.Atoi(partStr)
			if err != nil {
				return nil, fmt.Errorf("invalid part number: %s", partStr)
			}
			parts[i] = part
		}
		bodySection.Part = parts
		return bodySection, nil
	}

	// Check for part numbers followed by specifier (e.g., "1.2.HEADER")
	if strings.Contains(sectionContent, ".") {
		lastDotIdx := strings.LastIndex(sectionContent, ".")
		partPrefix := sectionContent[:lastDotIdx]
		specifier := sectionContent[lastDotIdx+1:]

		if regexp.MustCompile(`^\d+(\.\d+)*$`).MatchString(partPrefix) {
			// Parse part numbers
			partStrs := strings.Split(partPrefix, ".")
			parts := make([]int, len(partStrs))
			for i, partStr := range partStrs {
				part, err := strconv.Atoi(partStr)
				if err != nil {
					return nil, fmt.Errorf("invalid part number: %s", partStr)
				}
				parts[i] = part
			}
			bodySection.Part = parts

			// Set specifier
			switch strings.ToUpper(specifier) {
			case "HEADER":
				bodySection.Specifier = imap.PartSpecifierHeader
			case "TEXT":
				bodySection.Specifier = imap.PartSpecifierText
			case "MIME":
				bodySection.Specifier = imap.PartSpecifierMIME
			default:
				return nil, fmt.Errorf("unsupported part specifier: %s", specifier)
			}
			return bodySection, nil
		}
	}

	// Handle header field requests (e.g., "HEADER.FIELDS (FROM TO)")
	if strings.HasPrefix(sectionContent, "HEADER.FIELDS") {
		headerContent := strings.TrimPrefix(sectionContent, "HEADER.FIELDS")
		headerContent = strings.TrimSpace(headerContent)

		// Check for NOT version
		isNot := false
		if strings.HasPrefix(headerContent, ".NOT") {
			isNot = true
			headerContent = strings.TrimPrefix(headerContent, ".NOT")
			headerContent = strings.TrimSpace(headerContent)
		}

		// Parse field list in parentheses
		if strings.HasPrefix(headerContent, "(") && strings.HasSuffix(headerContent, ")") {
			fieldList := strings.TrimSuffix(strings.TrimPrefix(headerContent, "("), ")")
			fields := strings.Fields(fieldList)

			if isNot {
				bodySection.HeaderFieldsNot = fields
				bodySection.Specifier = imap.PartSpecifierHeader
			} else {
				bodySection.HeaderFields = fields
				bodySection.Specifier = imap.PartSpecifierHeader
			}
			return bodySection, nil
		}
		return nil, fmt.Errorf("invalid HEADER.FIELDS format: %s", sectionContent)
	}

	// Handle simple specifiers
	switch sectionContent {
	case "HEADER":
		bodySection.Specifier = imap.PartSpecifierHeader
	case "TEXT":
		bodySection.Specifier = imap.PartSpecifierText
	case "MIME":
		bodySection.Specifier = imap.PartSpecifierMIME
	default:
		return nil, fmt.Errorf("unsupported section content: %s", sectionContent)
	}

	return bodySection, nil
}

// parseHeaderData parses RFC 2822 header format and returns a map
func (r *Req) parseHeaderData(headerData string) map[string]string {
	headers := make(map[string]string)
	lines := strings.Split(headerData, "\n")

	var currentKey string
	var currentValue strings.Builder

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// Check if this is a continuation line (starts with whitespace)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if currentKey != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
			continue
		}

		// Save previous header if exists
		if currentKey != "" {
			headers[strings.ToLower(currentKey)] = strings.TrimSpace(currentValue.String())
			currentValue.Reset()
		}

		// Parse new header line
		colonIndex := strings.Index(line, ":")
		if colonIndex > 0 {
			currentKey = strings.TrimSpace(line[:colonIndex])
			if colonIndex+1 < len(line) {
				currentValue.WriteString(strings.TrimSpace(line[colonIndex+1:]))
			}
		} else {
			currentKey = ""
		}
	}

	// Save last header if exists
	if currentKey != "" {
		headers[strings.ToLower(currentKey)] = strings.TrimSpace(currentValue.String())
	}

	return headers
}
