package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"regexp"
	"strconv"
	"strings"
)

var reSpeedPrice = regexp.MustCompile(`\[(\d+)/`)

func round(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}

func applyBwMultiplier(v int) int {
	return int(round(float64(v)*cfg.Billing.BwMultiplier, 1))
}

func parseIsPatriot(tname string) bool {
	return strings.Contains(strings.ToLower(tname), "патриот города - ")
}

func parseSpeed(tcomment string) (int, error) {
	m := reSpeedPrice.FindStringSubmatch(tcomment)
	if len(m) > 1 {
		return strconv.Atoi(m[1])
	}
	return 0, nil
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
	md5c := md5.New()
	md5c.Write([]byte(dbr.Name))
	md5c.Write([]byte(strconv.Itoa(dbr.Speed)))
	md5c.Write([]byte(dbr.Comment))
	for _, ip := range dbr.Ips {
		md5c.Write([]byte(ip))
	}
	dbr.Hash = hex.EncodeToString(md5c.Sum(nil))
}

type Billing struct {
	dbTih *sql.DB
	dbKor *sql.DB
	cache []DbRec
}

func NewBilling() *Billing {
	log.Debug("init Billing ...")
	b := &Billing{
		dbTih: DbConnect(
			cfg.Billing.DB.Host,
			cfg.Billing.DB.Username,
			cfg.Billing.DB.Password,
			cfg.Billing.DB.DbNameTih),
		dbKor: DbConnect(
			cfg.Billing.DB.Host,
			cfg.Billing.DB.Username,
			cfg.Billing.DB.Password,
			cfg.Billing.DB.DbNameKor),
		cache: make([]DbRec, 0),
	}
	return b
}

func (b *Billing) Disconnect() {
	ehSkip(b.dbTih.Close())
	ehSkip(b.dbKor.Close())
}

func (b *Billing) dbOk() {
	var err error
	if err = b.dbTih.Ping(); err != nil {
		log.Warning("Tih DB ping failed, reconnect", err)
		if err = b.dbTih.Close(); err != nil {
			log.Warning("Tih DB close connection", err)
		}

		b.dbTih = DbConnect(cfg.Billing.DB.Host, cfg.Billing.DB.Username,
			cfg.Billing.DB.Password, cfg.Billing.DB.DbNameTih)
	}

	if err = b.dbKor.Ping(); err != nil {
		log.Warning("Kor DB ping failed, reconnect", err)
		if err = b.dbKor.Close(); err != nil {
			log.Warning("Kor DB close connection", err)
		}

		b.dbKor = DbConnect(cfg.Billing.DB.Host, cfg.Billing.DB.Username,
			cfg.Billing.DB.Password, cfg.Billing.DB.DbNameKor)
	}
}

func rowToDbRecord(dbrec *dbRow, cityCode string) (dbr DbRec) {
	var (
		ips     []string
		speed   int
		comment string
		err     error
	)

	speed, err = parseSpeed(dbrec.Tcomment)
	if err != nil {
		log.Warningf("can't parse speed from '%s'", dbrec.Tcomment)
		speed = 0
	} else {
		speed = applyBwMultiplier(speed)
	}

	if parseIsPatriot(dbrec.Tname) {
		comment = "PG"
	}

	ips = parseIps(dbrec.Ips)

	dbr = DbRec{
		Name:     dbrec.Name,
		Speed:    speed,
		Comment:  comment,
		Ips:      ips,
		CityCode: cityCode,
		Enabled:  dbrec.Enabled,
	}
	dbr.calcSum()

	return dbr
}

func (b *Billing) UpdateCache() {
	log.Debug("get users from Billing...")

	b.dbOk()
	b.cache = make([]DbRec, 0)
	q := cfg.Billing.GetUsersQuery

	rows, err := b.dbTih.Query(q)
	eh(err)

	var uinfo dbRow
	for rows.Next() {
		eh(rows.Scan(&uinfo.Name, &uinfo.Tname, &uinfo.Tcomment, &uinfo.Ips, &uinfo.Enabled))
		if IsInSlice(uinfo.Name, cfg.Shape.IgnoreIds) {
			uinfo.Tcomment = ""
		}
		b.cache = append(b.cache, rowToDbRecord(&uinfo, "tih"))
	}

	g := len(b.cache)
	log.Debugf("got %d records from Tih", g)

	rows, err = b.dbKor.Query(q)
	eh(err)

	for rows.Next() {
		eh(rows.Scan(&uinfo.Name, &uinfo.Tname, &uinfo.Tcomment, &uinfo.Ips, &uinfo.Enabled))
		if IsInSlice(uinfo.Name, cfg.Shape.IgnoreIds) {
			uinfo.Tcomment = ""
		}
		b.cache = append(b.cache, rowToDbRecord(&uinfo, "kor"))
	}
	log.Debugf("got %d records from Kor", len(b.cache)-g)
}

func (b *Billing) GetTariffInfo(tlId int) (speed int, comment string, err error) {
	b.dbOk()
	var stmt *sql.Stmt
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

	speed, err = parseSpeed(tcomment)
	if err != nil {
		log.Warningf("can't parse speed from '%s', set speed = 0", tcomment)
		speed = 0
	} else {
		speed = applyBwMultiplier(speed)
	}

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
			return &r
		}
	}
	return nil
}

func (b *Billing) FindByHash(h string) *DbRec {
	for _, r := range b.cache {
		if r.Hash == h {
			return &r
		}
	}
	return nil
}

func (b *Billing) FindByIP(ip string) *DbRec {
	for _, r := range b.cache {
		if IsInSlice(ip, r.Ips) {
			return &r
		}
	}
	return nil
}
