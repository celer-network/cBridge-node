package server

import (
	"database/sql"
	"fmt"
	"math/big"
	"time"

	cbn "github.com/celer-network/cBridge-go/cbridgenode"
	"github.com/celer-network/goutils/log"
	"github.com/celer-network/goutils/sqldb"
)

const (
	transferAllColumns             = "tid,txhash,chainid,token,transfertype,timelock,hashlock,status,relatedtid,relatedchainid,relatedtoken,amount,fee,transfergascost,confirmgascost,refundgascost,preimage,senderaddr,receiveraddr,txconfirmhash,txrefundhash,updatets,createts"
	transferAllColumnParams        = "$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23"
	timeLockSafeMargin             = 6 * time.Minute
	refundSafeMargin               = 3 * time.Minute
	maxPendingTimeOutRetryDuration = 15 * time.Minute
)

type DAL struct {
	*sqldb.Db
}

func NewDAL(driver, info string, poolSize int) (*DAL, error) {
	db, err := sqldb.NewDb(driver, info, poolSize)
	if err != nil {
		log.Errorf("fail with db init:%s, %s, %d, err:%+v", driver, info, poolSize, err)
		return nil, err
	}

	dal := &DAL{
		db,
	}
	return dal, nil
}

func (d *DAL) Close() {
	if d.Db != nil {
		d.Db.Close()
		d.Db = nil
	}
}

func closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Warnln("closeRows: error:", err)
	}
}

func (d *DAL) DB() *sqldb.Db {
	return d.Db
}

// The "monitor" table.
func (d *DAL) InsertMonitor(event string, blockNum uint64, blockIdx int64, restart bool) error {
	q := `INSERT INTO monitor (event, blocknum, blockidx, restart)
                VALUES ($1, $2, $3, $4)`
	res, err := d.Exec(q, event, blockNum, blockIdx, restart)
	return sqldb.ChkExec(res, err, 1, "InsertMonitor")
}

func (d *DAL) GetMonitorBlock(event string) (uint64, int64, bool, error) {
	var blockNum uint64
	var blockIdx int64
	q := `SELECT blocknum, blockidx FROM monitor WHERE event = $1`
	err := d.QueryRow(q, event).Scan(&blockNum, &blockIdx)
	found, err := sqldb.ChkQueryRow(err)
	return blockNum, blockIdx, found, err
}

func (d *DAL) UpdateMonitorBlock(event string, blockNum uint64, blockIdx int64) error {
	q := `UPDATE monitor SET blocknum = $1, blockidx = $2 WHERE event = $3`
	res, err := d.Exec(q, blockNum, blockIdx, event)
	return sqldb.ChkExec(res, err, 1, "UpdateMonitorBlock")
}

func (d *DAL) UpsertMonitorBlock(event string, blockNum uint64, blockIdx int64, restart bool) error {
	q := `INSERT INTO monitor (event, blocknum, blockidx, restart)
                VALUES ($1, $2, $3, $4) ON CONFLICT (event) DO UPDATE
                SET blocknum = excluded.blocknum, blockidx = excluded.blockidx`
	res, err := d.Exec(q, event, blockNum, blockIdx, restart)
	return sqldb.ChkExec(res, err, 1, "UpsertMonitorBlock")
}

type Transfer struct {
	TransferId      Hash
	TxHash          Hash
	ChainId         uint64
	Token           Addr
	TransferType    cbn.TransferType
	TimeLock        time.Time
	HashLock        Hash
	Status          cbn.TransferStatus
	RelatedTid      Hash
	RelatedChainId  uint64
	RelatedToken    Addr
	Amount          big.Int
	Fee             big.Int
	TransferGasCost big.Int
	ConfirmGasCost  big.Int
	RefundGasCost   big.Int
	Preimage        Hash
	Sender          Addr
	Receiver        Addr
	TxConfirmHash   Hash
	TxRefundHash    Hash
	UpdateTs        time.Time
	CreateTs        time.Time
}

func (d *DAL) InsertTransfer(tx *Transfer) error {
	tsNow := time.Now()
	q := fmt.Sprintf("INSERT INTO transfer (%s) VALUES (%s) ON CONFLICT DO NOTHING", transferAllColumns, transferAllColumnParams)
	_, err := d.Exec(q, tx.TransferId.String(), tx.TxHash.String(), tx.ChainId, tx.Token.String(), tx.TransferType,
		tx.TimeLock, tx.HashLock.String(), tx.Status, tx.RelatedTid.String(), tx.RelatedChainId, tx.RelatedToken.String(), tx.Amount.String(),
		tx.Fee.String(), tx.TransferGasCost.String(), tx.ConfirmGasCost.String(), tx.RefundGasCost.String(), "", tx.Sender.String(), tx.Receiver.String(), tx.TxConfirmHash.String(),
		tx.TxRefundHash.String(), tsNow, tsNow)
	return err
}

func (d *DAL) SetRelatedTxPreimage(preimage, relatedTid Hash) error {
	q := `UPDATE transfer SET preimage = $1 WHERE relatedtid = $2`
	_, err := d.Exec(q, preimage.String(), relatedTid.String())
	return err
}

func (d *DAL) ConfirmTransfer(tid, preimage, txConfirmHash Hash) error {
	q := `UPDATE transfer SET status = $1, preimage = $2, txconfirmhash = $3 WHERE tid = $4`
	_, err := d.Exec(q, cbn.TransferStatus_TRANSFER_STATUS_CONFIRMED, preimage.String(), txConfirmHash.String(), tid.String())
	return err
}

func (d *DAL) RefundTransfer(tid, txRefundHash Hash) error {
	q := `UPDATE transfer SET status = $1, txrefundhash = $2 WHERE tid = $3`
	_, err := d.Exec(q, cbn.TransferStatus_TRANSFER_STATUS_REFUNDED, txRefundHash.String(), tid.String())
	return err
}

func (d *DAL) RecordTransferIn(tid, txHash Hash) error {
	q := `UPDATE transfer SET status = $1, txhash = $2 WHERE tid = $3 and status in ($4,$5)`
	_, err := d.Exec(q, cbn.TransferStatus_TRANSFER_STATUS_LOCKED, txHash.String(), tid.String(),
		cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_START, cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_PENDING)
	return err
}

func (d *DAL) UpdateTransferStatus(tid Hash, to cbn.TransferStatus) error {
	q := `UPDATE transfer SET status = $1 WHERE tid = $2`
	_, err := d.Exec(q, to, tid.String())
	return err
}

func (d *DAL) SetTransferPreimage(tid Hash, preimage Hash) error {
	q := `UPDATE transfer SET status = $1 WHERE tid = $2`
	_, err := d.Exec(q, preimage.String(), tid.String())
	return err
}

func (d *DAL) SetTransferInAmountAndFee(tid Hash, amount, fee *big.Int) error {
	q := `UPDATE transfer SET amount = $1, fee = $2 WHERE tid = $3`
	_, err := d.Exec(q, amount.String(), fee.String(), tid.String())
	return err
}

func (d *DAL) SetPendingTransferIn(tid Hash, to, from cbn.TransferStatus) error {
	q := `UPDATE transfer SET status = $1, updatets = $2 WHERE tid = $3 and status = $4`
	_, err := d.Exec(q, to, time.Now(), tid.String(), from)
	return err
}

func (d *DAL) SetTransferStatusByFrom(tid Hash, to, from cbn.TransferStatus) error {
	q := `UPDATE transfer SET status = $1, updatets = $2 WHERE tid = $3 and status = $4`
	_, err := d.Exec(q, to, time.Now(), tid.String(), from)
	return err
}

func (d *DAL) GetTransferByTid(tid Hash) (*Transfer, bool, error) {
	tx := &Transfer{}
	q := fmt.Sprintf("SELECT %s from transfer where tid = $1", transferAllColumns)
	err := scanTransfer(d.QueryRow(q, tid.String()), tx)
	found, err := sqldb.ChkQueryRow(err)
	return tx, found, err
}

func (d *DAL) GetTransferByRelatedTid(relatedTid Hash) (*Transfer, bool, error) {
	tx := &Transfer{}
	q := fmt.Sprintf("SELECT %s from transfer where relatedtid = $1", transferAllColumns)
	err := scanTransfer(d.QueryRow(q, relatedTid.String()), tx)
	found, err := sqldb.ChkQueryRow(err)
	return tx, found, err
}

func (d *DAL) GetAllTransfers() ([]*Transfer, error) {
	q := fmt.Sprintf("SELECT %s from transfer", transferAllColumns)
	rows, err := d.Query(q)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetAllTransfersWithLimit(limit uint64) ([]*Transfer, error) {
	q := fmt.Sprintf("SELECT %s from transfer order by createts desc limit %d", transferAllColumns, limit)
	rows, err := d.Query(q)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetAllStartTransferIn() ([]*Transfer, error) {
	q := fmt.Sprintf("SELECT %s from transfer where status = $1 and timelock > $2 and transfertype = $3", transferAllColumns)
	rows, err := d.Query(q, cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_START, time.Now().Add(1*time.Hour), cbn.TransferType_TRANSFER_TYPE_IN)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetAllConfirmableLockedTransfer() ([]*Transfer, error) {
	q := fmt.Sprintf("SELECT %s from transfer where status = $1 and preimage is not null and preimage != ''", transferAllColumns)
	rows, err := d.Query(q, cbn.TransferStatus_TRANSFER_STATUS_LOCKED)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetRecoverTimeoutPendingTransferIn() ([]*Transfer, error) {
	// We find all pending transfers in which may have do transfer before 1 hour ago, but have not received the monitor.
	// We will try to send transfer in again for this transfer in. By set the status back to start from pending, the job of transfer in will try send again.
	// On another hand, if the transfer in time lock is expired, we will ignore this transfer in.
	tsNow := time.Now()
	q := fmt.Sprintf("SELECT %s from transfer where status = $1 and updatets < $2 and transfertype = $3 and timelock > $4", transferAllColumns)
	rows, err := d.Query(q, cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_PENDING, tsNow.Add(-1*maxPendingTimeOutRetryDuration), cbn.TransferType_TRANSFER_TYPE_IN, tsNow.Add(timeLockSafeMargin))
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetRecoverTimeoutPendingConfirm() ([]*Transfer, error) {
	tsNow := time.Now()
	q := fmt.Sprintf("SELECT %s from transfer where status = $1 and updatets < $2 and timelock > $3", transferAllColumns)
	rows, err := d.Query(q, cbn.TransferStatus_TRANSFER_STATUS_CONFIRM_PENDING, tsNow.Add(-1*maxPendingTimeOutRetryDuration), tsNow.Add(timeLockSafeMargin))
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetRecoverTimeoutPendingRefund() ([]*Transfer, error) {
	tsNow := time.Now()
	q := fmt.Sprintf("SELECT %s from transfer where status = $1 and updatets < $2", transferAllColumns)
	rows, err := d.Query(q, cbn.TransferStatus_TRANSFER_STATUS_REFUND_PENDING, tsNow.Add(-1*maxPendingTimeOutRetryDuration))
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func (d *DAL) GetAllRefundAbleTransferIn() ([]*Transfer, error) {
	q := fmt.Sprintf(`SELECT %s from transfer where status = $1 and timelock < $2 and transfertype = $3`, transferAllColumns)
	rows, err := d.Query(q, cbn.TransferStatus_TRANSFER_STATUS_LOCKED, time.Now(), cbn.TransferType_TRANSFER_TYPE_IN)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)
	var txs []*Transfer
	for rows.Next() {
		tx := &Transfer{}
		if err = scanTransfers(rows, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, err
}

func scanTransfers(rows *sql.Rows, tx *Transfer) error {
	var transferId, txHash, token, relatedToken, hashLock, relatedTid, amount, fee, transferFee, confirmFee, refundFee, preimage, sender, receiver, txConfirmHash, txRefundHash string
	err := rows.Scan(&transferId, &txHash, &tx.ChainId, &token, &tx.TransferType, &tx.TimeLock, &hashLock, &tx.Status,
		&relatedTid, &tx.RelatedChainId, &relatedToken, &amount, &fee, &transferFee, &confirmFee, &refundFee, &preimage, &sender,
		&receiver, &txConfirmHash, &txRefundHash, &tx.UpdateTs, &tx.CreateTs)
	if err != nil {
		return err
	}
	tx.TransferId = Hex2Hash(transferId)
	tx.TxHash = Hex2Hash(txHash)
	tx.Token = Hex2Addr(token)
	tx.RelatedToken = Hex2Addr(relatedToken)
	tx.HashLock = Hex2Hash(hashLock)
	tx.RelatedTid = Hex2Hash(relatedTid)
	tx.Amount.SetString(amount, 10)
	tx.Fee.SetString(fee, 10)
	tx.TransferGasCost.SetString(transferFee, 10)
	tx.ConfirmGasCost.SetString(confirmFee, 10)
	tx.RefundGasCost.SetString(refundFee, 10)
	tx.Preimage = Hex2Hash(preimage)
	tx.Sender = Hex2Addr(sender)
	tx.Receiver = Hex2Addr(receiver)
	tx.TxConfirmHash = Hex2Hash(txConfirmHash)
	tx.TxRefundHash = Hex2Hash(txRefundHash)
	return nil
}

func scanTransfer(row *sql.Row, tx *Transfer) error {
	var transferId, txHash, token, relatedToken, hashLock, relatedTid, amount, fee, transferFee, confirmFee, refundFee, preimage, sender, receiver, txConfirmHash, txRefundHash string
	err := row.Scan(&transferId, &txHash, &tx.ChainId, &token, &tx.TransferType, &tx.TimeLock, &hashLock, &tx.Status,
		&relatedTid, &tx.RelatedChainId, &relatedToken, &amount, &fee, &transferFee, &confirmFee, &refundFee, &preimage, &sender,
		&receiver, &txConfirmHash, &txRefundHash, &tx.UpdateTs, &tx.CreateTs)
	if err != nil {
		return err
	}
	tx.TransferId = Hex2Hash(transferId)
	tx.TxHash = Hex2Hash(txHash)
	tx.Token = Hex2Addr(token)
	tx.RelatedToken = Hex2Addr(relatedToken)
	tx.HashLock = Hex2Hash(hashLock)
	tx.RelatedTid = Hex2Hash(relatedTid)
	tx.Amount.SetString(amount, 10)
	tx.Fee.SetString(fee, 10)
	tx.TransferGasCost.SetString(transferFee, 10)
	tx.ConfirmGasCost.SetString(confirmFee, 10)
	tx.RefundGasCost.SetString(refundFee, 10)
	tx.Preimage = Hex2Hash(preimage)
	tx.Sender = Hex2Addr(sender)
	tx.Receiver = Hex2Addr(receiver)
	tx.TxConfirmHash = Hex2Hash(txConfirmHash)
	tx.TxRefundHash = Hex2Hash(txRefundHash)
	return nil
}
