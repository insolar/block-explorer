package migrations

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/jinzhu/gorm"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/etl/models"
	"github.com/insolar/block-explorer/instrumentation/converter"
)

// GenerateRandBytesLen generates random bytes array with len
func GenerateRandBytesLen(l int) ([]byte, error) {
	b := make([]byte, l)
	if _, err := crand.Read(b); err != nil {
		return []byte{}, err
	}
	return b, nil
}

func GeneratePulses(amount int) []models.Pulse {
	tNow := time.Now().Unix()
	startPulse := 4000000
	var pulses []models.Pulse
	for i := startPulse; i < startPulse+amount; i++ {
		pulses = append(pulses,
			models.Pulse{
				PulseNumber:     int64(i),
				PrevPulseNumber: int64(i) - 1,
				NextPulseNumber: int64(i) + 1,
				IsComplete:      true,
				IsSequential:    true,
				Timestamp:       tNow + int64(i*10),
			})
	}
	return pulses
}

func notNullJetID() string {
	for {
		jetID := gen.JetID()
		id := binary.BigEndian.Uint64(jetID.Prefix())
		if id == 0 {
			continue
		}
		return converter.JetIDToString(jetID)
	}
}

func GenerateJetDrops(pulses []models.Pulse, amount int) ([]models.JetDrop, error) {
	tNow := time.Now().Unix()
	var jDrops []models.JetDrop
	for i := 1; i < amount; i++ {
		rawData, err := GenerateRandBytesLen(32)
		if err != nil {
			return []models.JetDrop{}, err
		}
		randPulseNum := rand.Intn(len(pulses))
		rPnum := pulses[randPulseNum].PulseNumber
		pn := insolar.PulseNumber(rPnum)
		jID := notNullJetID()
		jDrops = append(jDrops, models.JetDrop{
			JetID:          jID,
			PulseNumber:    rPnum,
			FirstPrevHash:  gen.IDWithPulse(pn).Bytes(),
			SecondPrevHash: gen.IDWithPulse(pn).Bytes(),
			Hash:           rawData,
			RawData:        rawData,
			Timestamp:      tNow + int64(i*2),
			RecordAmount:   100,
		})
	}
	return jDrops, nil
}

func GenerateRecords(jDrops []models.JetDrop, amount int) ([]models.Record, error) {
	tNow := time.Now().Unix()
	var records []models.Record
	for i := 1; i < amount; i++ {
		rawData, err := GenerateRandBytesLen(32)
		if err != nil {
			return []models.Record{}, err
		}
		randJetID := rand.Intn(len(jDrops))
		randJet := jDrops[randJetID].JetID
		jetPulseNum := jDrops[randJetID].PulseNumber
		pn := insolar.PulseNumber(jetPulseNum)
		records = append(records, models.Record{
			Reference:           gen.IDWithPulse(pn).Bytes(),
			Type:                "state",
			ObjectReference:     gen.IDWithPulse(pn).Bytes(),
			PrototypeReference:  gen.IDWithPulse(pn).Bytes(),
			Payload:             rawData,
			PrevRecordReference: gen.IDWithPulse(pn).Bytes(),
			Hash:                rawData,
			RawData:             rawData,
			JetID:               randJet,
			PulseNumber:         jetPulseNum,
			Order:               0,
			Timestamp:           tNow + int64(i*2),
		})
	}
	return records, nil
}

func generateData(tx *gorm.DB, cfg *configuration.TestDB) error {
	pulses := GeneratePulses(cfg.TestPulses)
	for _, p := range pulses {
		pulse := p
		if err := tx.Save(&pulse).Error; err != nil {
			return err
		}
	}
	jdrops, err := GenerateJetDrops(pulses, cfg.TestJetDrops)
	if err != nil {
		return err
	}
	for _, jd := range jdrops {
		jetDrop := jd
		if err := tx.Save(&jetDrop).Error; err != nil {
			return err
		}
	}
	records, err := GenerateRecords(jdrops, cfg.TestRecords)
	if err != nil {
		return err
	}
	for _, rec := range records {
		record := rec
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
	}
	return nil
}
