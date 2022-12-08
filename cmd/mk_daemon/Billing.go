package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

func applyBwMultiplier(v int) int {
	return int(float64(v) * CFG.Billing.BwMultiplier)
}

func DbConnect(hostname string, username string, password string, dbname string) *sql.DB {
	conn, err := sql.Open(formatDsn(hostname, username, password, dbname))
	errMsg := fmt.Sprintf("Can't connect to DB `%s` at `%s` as `%s`\n", dbname, hostname, username)
	eh(err, errMsg)

	eh(conn.Ping(), errMsg)
	return conn
}

func formatDsn(hostname string, username string, password string, dbname string) (string, string) {
	var suffix string

	if hostname == "localhost" {
		suffix = "/" + dbname
	} else {
		suffix = fmt.Sprintf("tcp(%s:3306)/%s", hostname, dbname)
	}

	return "mysql", fmt.Sprintf("%s:%s@%s", username, password, suffix)
}

type (
	DbRec struct {
		Name     string   // "mserg-hm#3780#5693"
		Speed    int      // 42
		Comment  string   // "PG" for 'patriot' tariffs
		Ips      []string // ["192.168.62.254", "192.168.62.143"]
		Hash     string   // md5 of Name + Speed + Comment + Ips
		CityCode string   // "tih"
		Enabled  bool     // true
	}

	dbRow struct {
		Name     string
		Tname    string
		Tcomment string
		Ips      string
		Enabled  bool
	}
)

func (dbr *DbRec) calcSum() {
	h := fnv.New128()
	h.Write([]byte(dbr.Name))
	h.Write([]byte(strconv.Itoa(dbr.Speed)))
	h.Write([]byte(dbr.Comment))
	for _, ip := range dbr.Ips {
		h.Write([]byte(ip))
	}
	dbr.Hash = hex.EncodeToString(h.Sum(nil))
}

type Billing struct {
	dbTih *sql.DB
	dbKor *sql.DB
	cache []*DbRec
}

func NewBilling() *Billing {
	LOG.Debug("init Billing ...")

	// todo: this is ugly
	b := &Billing{
		dbTih: DbConnect(
			CFG.Billing.DB.Host,
			CFG.Billing.DB.Username,
			CFG.Billing.DB.Password,
			CFG.Billing.DB.DbNameTih),
		dbKor: DbConnect(
			CFG.Billing.DB.Host,
			CFG.Billing.DB.Username,
			CFG.Billing.DB.Password,
			CFG.Billing.DB.DbNameKor),
		cache: make([]*DbRec, 0),
	}
	return b
}

func (b *Billing) Disconnect() {
	// todo: mhe 🤮
	ehSkip(b.dbTih.Close())
	ehSkip(b.dbKor.Close())
}

func (b *Billing) dbOk() {
	var err error
	if err = b.dbTih.Ping(); err != nil {
		LOG.Warning("Tih DB ping failed, reconnect", err)
		if err = b.dbTih.Close(); err != nil {
			LOG.Warning("Tih DB close connection", err)
		}

		b.dbTih = DbConnect(CFG.Billing.DB.Host, CFG.Billing.DB.Username,
			CFG.Billing.DB.Password, CFG.Billing.DB.DbNameTih)
	}

	if err = b.dbKor.Ping(); err != nil {
		LOG.Warning("Kor DB ping failed, reconnect", err)
		if err = b.dbKor.Close(); err != nil {
			LOG.Warning("Kor DB close connection", err)
		}

		b.dbKor = DbConnect(CFG.Billing.DB.Host, CFG.Billing.DB.Username,
			CFG.Billing.DB.Password, CFG.Billing.DB.DbNameKor)
	}
}

func rowToDbRecord(dbRec *dbRow, cityCode string) (dbr *DbRec) {
	var (
		ips     []string
		comment string
	)

	if parseIsPatriot(dbRec.Tname) {
		comment = "PG"
	}

	ips = parseIps(dbRec.Ips)

	dbr = &DbRec{
		Name:     dbRec.Name,
		Speed:    applyBwMultiplier(parseSpeed(dbRec.Tcomment)),
		Comment:  comment,
		Ips:      ips,
		CityCode: cityCode,
		Enabled:  dbRec.Enabled,
	}
	dbr.calcSum()

	return dbr
}

func (b *Billing) UpdateCache() {
	LOG.Debug("get users from Billing...")

	b.dbOk()
	b.cache = make([]*DbRec, 0)
	q := CFG.Billing.GetUsersQuery

	rows, err := b.dbTih.Query(q)
	eh(err)

	var uinfo dbRow
	for rows.Next() {
		eh(rows.Scan(&uinfo.Name, &uinfo.Tname, &uinfo.Tcomment, &uinfo.Ips, &uinfo.Enabled))
		if IsInSlice(uinfo.Name, CFG.Shape.IgnoreIds) {
			uinfo.Tcomment = ""
		}
		b.cache = append(b.cache, rowToDbRecord(&uinfo, "tih"))
	}

	g := len(b.cache)
	LOG.Debugf("got %d records from Tih", g)

	rows, err = b.dbKor.Query(q)
	eh(err)

	for rows.Next() {
		eh(rows.Scan(&uinfo.Name, &uinfo.Tname, &uinfo.Tcomment, &uinfo.Ips, &uinfo.Enabled))
		if IsInSlice(uinfo.Name, CFG.Shape.IgnoreIds) {
			uinfo.Tcomment = ""
		}
		b.cache = append(b.cache, rowToDbRecord(&uinfo, "kor"))
	}
	LOG.Debugf("got %d records from Kor", len(b.cache)-g)
}

func (b *Billing) GetTariffInfo(tlId int) (speed int, comment string, err error) {
	b.dbOk()
	var stmt *sql.Stmt
	//goland:noinspection SqlNoDataSourceInspection,SqlResolve
	stmt, err = b.dbTih.Prepare(`
	select t.name, t.comments
	from account_tariff_link as atl
	join tariffs as t on atl.tariff_id = t.id
	where atl.id = ? and t.comments like '%[%/%]%'
	limit 1
	`)
	if err != nil {
		return 0, "", err
	}

	var tname, tcomment string
	err = stmt.QueryRow(tlId).Scan(&tname, &tcomment)
	if err != nil {
		return 0, "", err
	}

	speed = applyBwMultiplier(parseSpeed(tcomment))

	if parseIsPatriot(tname) {
		comment = "PG"
	} else {
		comment = ""
	}
	return
}

func (b *Billing) FindByName(name string) *DbRec {
	for _, r := range b.cache {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func (b *Billing) FindByHash(h string) *DbRec {
	for _, r := range b.cache {
		if r.Hash == h {
			return r
		}
	}
	return nil
}

func (b *Billing) FindByIP(ip string) *DbRec {
	for _, r := range b.cache {
		if IsInSlice(ip, r.Ips) {
			return r
		}
	}
	return nil
}
