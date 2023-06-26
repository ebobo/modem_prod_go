package sqlitestore

import (
	"log"

	"github.com/ebobo/modem_prod_go/pkg/model"
)

func (s *SqliteStore) AddModem(modem model.Modem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.NamedExec(
		`INSERT INTO modems (
			mac_address,
			ipv6,
			switch_port,
			model,
			state,
			firmware,
			serial,
			kernel,
			upgraded,
			last_updated,
			fail_count,
			sim_provider,
			sim_status,
			imei,
			iccid,
			imsi,
			progress)
		 VALUES(
			:mac_address,
			:ipv6,
			:switch_port,
			:model,
			:state,
			:firmware,
			:serial,
			:kernel,
			:upgraded,
			:last_updated,
			:fail_count,
			:sim_provider,
			:sim_status,
			:imei,
			:iccid,
			:imsi,
			:progress)`, modem)
	return err
}

func (s *SqliteStore) GetModem(mac string) (model.Modem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var modem model.Modem
	return modem, s.db.QueryRowx("SELECT * FROM modems WHERE mac_address = ?", mac).StructScan(&modem)
}

// 0: unknown, 1: normal, 2: busy, 3: error
func (s *SqliteStore) SetModemState(mac string, state int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CheckForZeroRowsAffected(s.db.Exec("UPDATE modems SET state = ? WHERE mac_address = ?", state, mac))
}

// progress is a int from 0 to 100
func (s *SqliteStore) SetModemUpgradeProgress(mac string, progress int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CheckForZeroRowsAffected(s.db.Exec("UPDATE modems SET progress = ? WHERE mac_address = ?", progress, mac))
}

func (s *SqliteStore) UpdateModem(modem model.Modem) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return CheckForZeroRowsAffected(s.db.NamedExec(
		`UPDATE modems SET 
			ipv6 = :ipv6, 
			switch_port = :switch_port,
			model = :model,
			state = :state,
			firmware = :firmware,
			serial = :serial,
			kernel = :kernel,
			upgraded = :upgraded,
			last_updated = :last_updated,
			fail_count = :fail_count,
			sim_provider = :sim_provider,
			sim_status = :sim_status,
			imei = :imei,
			iccid = :iccid,
			imsi = :imsi,
			progress = :progress
		WHERE 
			mac_address = :mac_address`, modem))
}

func (s *SqliteStore) ListModems() ([]model.Modem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var modems []model.Modem
	return modems, s.db.Select(&modems, "SELECT * FROM modems")
}

func (s *SqliteStore) DeleteModem(mac string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := s.db.Exec("DELETE FROM modems WHERE mac_address = ?", mac)
	return err
}

func (s *SqliteStore) PrintModems() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Println("Printing modems")

	row, err := s.db.Query("SELECT * FROM modems ORDER BY mac_address")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()
	for row.Next() {
		var modem model.Modem
		row.Scan(&modem)
		log.Println("Modem: ", modem)
	}
	return nil
}
