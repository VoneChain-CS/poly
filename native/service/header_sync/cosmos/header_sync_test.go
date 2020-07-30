/*
 * Copyright (C) 2020 The poly network Authors
 * This file is part of The poly network library.
 *
 * The  poly network  is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The  poly network  is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 * You should have received a copy of the GNU Lesser General Public License
 * along with The poly network .  If not, see <http://www.gnu.org/licenses/>.
 */

package cosmos

import (
	"encoding/hex"
	"fmt"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/polynetwork/poly/account"
	"github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/core/genesis"
	"github.com/polynetwork/poly/core/store/leveldbstore"
	"github.com/polynetwork/poly/core/store/overlaydb"
	"github.com/polynetwork/poly/core/types"
	"github.com/polynetwork/poly/native"
	scom "github.com/polynetwork/poly/native/service/header_sync/common"
	"github.com/polynetwork/poly/native/storage"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	SUCCESS = iota
	GENESIS_PARAM_ERROR
	GENESIS_INITIALIZED
	SYNCBLOCK_PARAM_ERROR
	SYNCBLOCK_ORPHAN
	DIFFICULTY_ERROR
	NONCE_ERROR
	UNKNOWN
)

var (
	acct     *account.Account = account.NewAccount("")
	setBKers                  = func() {
		genesis.GenesisBookkeepers = []keypair.PublicKey{acct.PublicKey}
	}
)

func init() {
	setBKers()
}

func typeOfError(e error) int {
	if e == nil {
		return SUCCESS
	}
	errDesc := e.Error()
	if strings.Contains(errDesc, "ETHHandler GetHeaderByHeight, genesis header had been initialized") {
		return GENESIS_INITIALIZED
	} else if strings.Contains(errDesc, "ETHHandler SyncGenesisHeader: getGenesisHeader, deserialize header err:") {
		return GENESIS_PARAM_ERROR
	} else if strings.Contains(errDesc, "SyncBlockHeader, deserialize header err:") {
		return SYNCBLOCK_PARAM_ERROR
	} else if strings.Contains(errDesc, "SyncBlockHeader, get the parent block failed. Error:") {
		return SYNCBLOCK_ORPHAN
	} else if strings.Contains(errDesc, "SyncBlockHeader, invalid difficulty:") {
		return DIFFICULTY_ERROR
	} else if strings.Contains(errDesc, "SyncBlockHeader, verify header error:") {
		return NONCE_ERROR
	}
	return UNKNOWN
}

func NewNative(args []byte, tx *types.Transaction, db *storage.CacheDB) *native.NativeService {
	if db == nil {
		store, _ := leveldbstore.NewMemLevelDBStore()
		db = storage.NewCacheDB(overlaydb.NewOverlayDB(store))
	}
	ns, _ := native.NewNativeService(db, tx, 0, 0, common.Uint256{0}, 0, args, false)
	return ns
}

func TestSyncGenesisHeader(t *testing.T) {
	header10000, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718904e220c08eae29ff70510ffc5bfd5022a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e3220e728973f68379b28c499cfa4a6234a79ca6e7579290fa0c5013a65ce7af69db2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a2021b3a41b885e7248c813ee0290f540cd6a1227d21e263a875497a6aa7972e4ec72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108904e1a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb8092268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08efe29ff70510cd98fefe022240e3807de6d7d219a0c2ca6187b6a447711de395f8a51053ed7fde07f522b1977987411d74128a31a03c0df91790b10596776fe5381363706086ca4086cb96ca061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
	param := new(scom.SyncGenesisHeaderParam)
	param.ChainID = 5
	param.GenesisHeader = header10000
	sink := common.NewZeroCopySink(nil)
	param.Serialization(sink)

	tx := &types.Transaction{
		SignedAddr: []common.Address{acct.Address},
	}

	native := NewNative(sink.Bytes(), tx, nil)
	cosmosHandler := NewCosmosHandler()
	err := cosmosHandler.SyncGenesisHeader(native)
	if err != nil {
		fmt.Printf("err: %s", err.Error())
	}
	assert.Equal(t, SUCCESS, typeOfError(err))

	info, _ := GetEpochSwitchInfo(native, param.ChainID)
	assert.Equal(t, info != nil, true)
	assert.Equal(t, uint64(info.Height), uint64(10000))
}

func TestSyncBlockHeader(t *testing.T) {
	cosmosHandler := NewCosmosHandler()
	var native *native.NativeService
	{
		header10000, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718904e220c08eae29ff70510ffc5bfd5022a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e3220e728973f68379b28c499cfa4a6234a79ca6e7579290fa0c5013a65ce7af69db2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a2021b3a41b885e7248c813ee0290f540cd6a1227d21e263a875497a6aa7972e4ec72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108904e1a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb8092268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08efe29ff70510cd98fefe022240e3807de6d7d219a0c2ca6187b6a447711de395f8a51053ed7fde07f522b1977987411d74128a31a03c0df91790b10596776fe5381363706086ca4086cb96ca061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncGenesisHeaderParam)
		param.ChainID = 5
		param.GenesisHeader = header10000
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		tx := &types.Transaction{
			SignedAddr: []common.Address{acct.Address},
		}

		native = NewNative(sink.Bytes(), tx, nil)
		err := cosmosHandler.SyncGenesisHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		info, _ := GetEpochSwitchInfo(native, param.ChainID)
		assert.Equal(t, info != nil, true)
		assert.Equal(t, uint64(info.Height), uint64(10000))
	}
	{
		header10001, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718914e220c08efe29ff70510cd98fefe022a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb80932200622a4189db86c59caf2e5c993001125147821024fa1b4483ab8bfd9a5ab516c422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a200cee95b828cd74a3078db7c6e3ef8e75c282edf5396b525b831309632edc7a4a72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108914e1a480a208e0b7bb3f66dc7e215ffbf77c9971cfd12742bf495bbc92b38b6fdbf0a9c235f122408011220b27ce0ec06318b860c6b66b3a6ed10c49034532aaf159de509278cbf949969412268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08f4e29ff70510edfbe0a603224012da471ec08ccd347b40f5c505054bc49b28dea720cb38e84c80a6fcb3042148a374e1a179b3b20064dc35aead710a6e8600a8e3b0e0823384801129924725061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10002, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718924e220c08f4e29ff70510edfbe0a6032a480a208e0b7bb3f66dc7e215ffbf77c9971cfd12742bf495bbc92b38b6fdbf0a9c235f122408011220b27ce0ec06318b860c6b66b3a6ed10c49034532aaf159de509278cbf9499694132203bd019d6ef8763f689bd053b3d5d96c99d096456e862af5dff0382c8dfab9483422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20e22606949c1bf971360a980bf85a0cc9ec7753ad02f4258f1998a4194ba78a2872146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108924e1a480a20ee52cb572507c078193a7e1eba63b4ff50f3f3cd2d91e77d9a1da13764ae87b512240801122090bc1b8a3706428b1ea03f8d4466c71832f8c2f0affaa604d67ef726e2d46e8b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08f9e29ff70510bebbd6d40322406254e69b1a46a928a0815afe9a0175d7f7f2831fe2b166a4e9c9c4f58668d69abbd6638eef817fd2a06b6f03334272e636b576017a66956895c60fda73a2e5091a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10001)
		param.Headers = append(param.Headers, header10002)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		assert.Error(t, err)
	}
}

/*
insert a new block
*/
func TestSyncBlockHeader2(t *testing.T) {
	cosmosHandler := NewCosmosHandler()
	var native *native.NativeService
	{
		header10000, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718904e220c08eae29ff70510ffc5bfd5022a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e3220e728973f68379b28c499cfa4a6234a79ca6e7579290fa0c5013a65ce7af69db2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a2021b3a41b885e7248c813ee0290f540cd6a1227d21e263a875497a6aa7972e4ec72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108904e1a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb8092268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08efe29ff70510cd98fefe022240e3807de6d7d219a0c2ca6187b6a447711de395f8a51053ed7fde07f522b1977987411d74128a31a03c0df91790b10596776fe5381363706086ca4086cb96ca061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncGenesisHeaderParam)
		param.ChainID = 5
		param.GenesisHeader = header10000
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		tx := &types.Transaction{
			SignedAddr: []common.Address{acct.Address},
		}

		native = NewNative(sink.Bytes(), tx, nil)
		err := cosmosHandler.SyncGenesisHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		info, _ := GetEpochSwitchInfo(native, param.ChainID)
		assert.Equal(t, info != nil, true)
		assert.Equal(t, uint64(info.Height), uint64(10000))
	}
	{
		header10010, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189a4e220c089de39ff70510ddf7abe5022a480a20520e78552c565e47a32fae7d923e61e33565a1518952cfadd767d0cc763c1ba5122408011220f65f6b4edd4b7f52bd0ac8bec534a646d68d152021a391b387169a169194635232201450e9e06022e395b9176448a4718f2c1f161ce3cfd501205245f19686038765422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a200f4cb2e3e6b3437ff1810c3420884ffbe13a8826c2830d7bec2c586c95a33f1572146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089a4e1a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a2e39ff70510a5fe848d032240bdbb92c73dd1bd5a61ebc72099b76d3d934573a5a54196a04e87e010964864d34e2a2a42ada217c5c9be210adc1cd0b2091c32fb9525fdabfb247fd7660466001a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10011, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189b4e220c08a2e39ff70510a5fe848d032a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c322055b77f9a764871721521ff6aeb920fe01bc440446712b483866cd4d09944839e422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20b4cf9c1ec8e800ee5e201678c25580124ea8ae1e3dc7c4128d67b92ac876522f72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089b4e1a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a7e39ff70510ecbbe6b903224048b1c9b7b32bdc0ecc11cb491dcd23f621bb799969221dae82e28645d229310eac1e3cde1aae6056b395d75b4d3adc96190c3cc40dc1ecaa2e431f4d5b78620b1a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10010)
		param.Headers = append(param.Headers, header10011)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		assert.Error(t, err)
		assert.Equal(t, SUCCESS, typeOfError(err))

		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10011))
	}
	{
		header10001, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718914e220c08efe29ff70510cd98fefe022a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb80932200622a4189db86c59caf2e5c993001125147821024fa1b4483ab8bfd9a5ab516c422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a200cee95b828cd74a3078db7c6e3ef8e75c282edf5396b525b831309632edc7a4a72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108914e1a480a208e0b7bb3f66dc7e215ffbf77c9971cfd12742bf495bbc92b38b6fdbf0a9c235f122408011220b27ce0ec06318b860c6b66b3a6ed10c49034532aaf159de509278cbf949969412268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08f4e29ff70510edfbe0a603224012da471ec08ccd347b40f5c505054bc49b28dea720cb38e84c80a6fcb3042148a374e1a179b3b20064dc35aead710a6e8600a8e3b0e0823384801129924725061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10002, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718924e220c08f4e29ff70510edfbe0a6032a480a208e0b7bb3f66dc7e215ffbf77c9971cfd12742bf495bbc92b38b6fdbf0a9c235f122408011220b27ce0ec06318b860c6b66b3a6ed10c49034532aaf159de509278cbf9499694132203bd019d6ef8763f689bd053b3d5d96c99d096456e862af5dff0382c8dfab9483422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20e22606949c1bf971360a980bf85a0cc9ec7753ad02f4258f1998a4194ba78a2872146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108924e1a480a20ee52cb572507c078193a7e1eba63b4ff50f3f3cd2d91e77d9a1da13764ae87b512240801122090bc1b8a3706428b1ea03f8d4466c71832f8c2f0affaa604d67ef726e2d46e8b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08f9e29ff70510bebbd6d40322406254e69b1a46a928a0815afe9a0175d7f7f2831fe2b166a4e9c9c4f58668d69abbd6638eef817fd2a06b6f03334272e636b576017a66956895c60fda73a2e5091a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10001)
		param.Headers = append(param.Headers, header10002)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10011))
	}
}

/*
sync block before genensis
*/
func TestSyncBlockHeader3(t *testing.T) {
	cosmosHandler := NewCosmosHandler()
	var native *native.NativeService
	{
		header10000, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718904e220c08eae29ff70510ffc5bfd5022a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e3220e728973f68379b28c499cfa4a6234a79ca6e7579290fa0c5013a65ce7af69db2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a2021b3a41b885e7248c813ee0290f540cd6a1227d21e263a875497a6aa7972e4ec72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108904e1a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb8092268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08efe29ff70510cd98fefe022240e3807de6d7d219a0c2ca6187b6a447711de395f8a51053ed7fde07f522b1977987411d74128a31a03c0df91790b10596776fe5381363706086ca4086cb96ca061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncGenesisHeaderParam)
		param.ChainID = 5
		param.GenesisHeader = header10000
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		tx := &types.Transaction{
			SignedAddr: []common.Address{acct.Address},
		}

		native = NewNative(sink.Bytes(), tx, nil)
		err := cosmosHandler.SyncGenesisHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//has, _ := hasGenesis(native, param.ChainID)
		//assert.Equal(t, has, true)
		//
		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10000))
	}
	{
		header9999, _ := hex.DecodeString("0aa8020a02080a120774657374696e67188f4e220c08e5e29ff70510abdb8ea7022a480a20727d85a7eb9a24c6fa1916456049197ba51046f84ee7270a00996f09206402fa1224080112206b044fe89ef52ac7977a9bd2569d49b47b3a5d895244fcb283fc875220fb5e063220a70dd9071d2ecac99ce7eeb462c7e56bcc99fab3cb52dd45cfe2502313777f55422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a209658cd0c9251c2514753fd85c0ebf649f8bec6bf885813dccbada1a575b2156c72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701088f4e1a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08eae29ff70510ffc5bfd5022240912bf904c00e06e8e922459179ee611d13082a10efefeb08ff238dae6d5f0c4f25034e70079c48709c1d3c0849a4fc2470743e871982dc860a94f2a68d07520c1a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header9999)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		assert.Error(t, err)
	}
}

func TestSyncBlockHeaderTwice(t *testing.T) {
	cosmosHandler := NewCosmosHandler()
	var native *native.NativeService
	{
		header10000, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718904e220c08eae29ff70510ffc5bfd5022a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e3220e728973f68379b28c499cfa4a6234a79ca6e7579290fa0c5013a65ce7af69db2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a2021b3a41b885e7248c813ee0290f540cd6a1227d21e263a875497a6aa7972e4ec72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108904e1a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb8092268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08efe29ff70510cd98fefe022240e3807de6d7d219a0c2ca6187b6a447711de395f8a51053ed7fde07f522b1977987411d74128a31a03c0df91790b10596776fe5381363706086ca4086cb96ca061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncGenesisHeaderParam)
		param.ChainID = 5
		param.GenesisHeader = header10000
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		tx := &types.Transaction{
			SignedAddr: []common.Address{acct.Address},
		}

		native = NewNative(sink.Bytes(), tx, nil)
		err := cosmosHandler.SyncGenesisHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//has, _ := hasGenesis(native, param.ChainID)
		//assert.Equal(t, has, true)
		//
		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10000))
	}
	{
		header10011, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189b4e220c08a2e39ff70510a5fe848d032a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c322055b77f9a764871721521ff6aeb920fe01bc440446712b483866cd4d09944839e422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20b4cf9c1ec8e800ee5e201678c25580124ea8ae1e3dc7c4128d67b92ac876522f72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089b4e1a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a7e39ff70510ecbbe6b903224048b1c9b7b32bdc0ecc11cb491dcd23f621bb799969221dae82e28645d229310eac1e3cde1aae6056b395d75b4d3adc96190c3cc40dc1ecaa2e431f4d5b78620b1a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10010, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189a4e220c089de39ff70510ddf7abe5022a480a20520e78552c565e47a32fae7d923e61e33565a1518952cfadd767d0cc763c1ba5122408011220f65f6b4edd4b7f52bd0ac8bec534a646d68d152021a391b387169a169194635232201450e9e06022e395b9176448a4718f2c1f161ce3cfd501205245f19686038765422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a200f4cb2e3e6b3437ff1810c3420884ffbe13a8826c2830d7bec2c586c95a33f1572146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089a4e1a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a2e39ff70510a5fe848d032240bdbb92c73dd1bd5a61ebc72099b76d3d934573a5a54196a04e87e010964864d34e2a2a42ada217c5c9be210adc1cd0b2091c32fb9525fdabfb247fd7660466001a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10010)
		param.Headers = append(param.Headers, header10011)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10011))
	}
	{
		header10011, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189b4e220c08a2e39ff70510a5fe848d032a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c322055b77f9a764871721521ff6aeb920fe01bc440446712b483866cd4d09944839e422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20b4cf9c1ec8e800ee5e201678c25580124ea8ae1e3dc7c4128d67b92ac876522f72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089b4e1a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a7e39ff70510ecbbe6b903224048b1c9b7b32bdc0ecc11cb491dcd23f621bb799969221dae82e28645d229310eac1e3cde1aae6056b395d75b4d3adc96190c3cc40dc1ecaa2e431f4d5b78620b1a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10012, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189c4e220c08a7e39ff70510ecbbe6b9032a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b32202496df827069967ea78dd93d07e6389e95abadd81cb5bd53cb9048b73a7ae0f2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a206a1f58abdc0c358ed42023b6da978c6baa7c748f2534f42cec212c5ddeddd16172146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b601089c4e1a480a20744cf79eec130b744c9325ee44eb74661cfbe090989ee27af6a6856750ff41751224080112204a684ab601bb25ede19c18622513ecd95637dff4ef6270a1cec154e35a939a212267080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0b08ade39ff70510e9cad70b2240d448c0de45611b935f1f4d37ca33cafa174c9eebdc074febdf70eaac9fc384bf669369c46ed279b18f5bd7a73999d4ae46c382204947f019fa3fca1ea9d0dc021a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10011)
		param.Headers = append(param.Headers, header10012)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10012))
	}
}

func TestSyncBlockHeaderUnorder(t *testing.T) {
	cosmosHandler := NewCosmosHandler()
	var native *native.NativeService
	{
		header10000, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718904e220c08eae29ff70510ffc5bfd5022a480a20a952550d320e34bd910fdf47d4b0d1b9b2c691d0432ebe5c2f25174c484663781224080112206fc745af7c59194f8e841a4e8bf89e6b7e873d1dfb807829deafc1093fcbe33e3220e728973f68379b28c499cfa4a6234a79ca6e7579290fa0c5013a65ce7af69db2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a2021b3a41b885e7248c813ee0290f540cd6a1227d21e263a875497a6aa7972e4ec72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108904e1a480a20d7147608a68aa6e72f26d196dd5ea13ab1e580eee72cb1d84b3e6e49c5a5ffd3122408011220e9dc35792711de4bb8b19ecb7ce888cbd0835bec6b16d7dfc33fe4667bccb8092268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08efe29ff70510cd98fefe022240e3807de6d7d219a0c2ca6187b6a447711de395f8a51053ed7fde07f522b1977987411d74128a31a03c0df91790b10596776fe5381363706086ca4086cb96ca061a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncGenesisHeaderParam)
		param.ChainID = 5
		param.GenesisHeader = header10000
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		tx := &types.Transaction{
			SignedAddr: []common.Address{acct.Address},
		}

		native = NewNative(sink.Bytes(), tx, nil)
		err := cosmosHandler.SyncGenesisHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//has, _ := hasGenesis(native, param.ChainID)
		//assert.Equal(t, has, true)
		//
		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10000))
	}
	{
		header10010, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189a4e220c089de39ff70510ddf7abe5022a480a20520e78552c565e47a32fae7d923e61e33565a1518952cfadd767d0cc763c1ba5122408011220f65f6b4edd4b7f52bd0ac8bec534a646d68d152021a391b387169a169194635232201450e9e06022e395b9176448a4718f2c1f161ce3cfd501205245f19686038765422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a200f4cb2e3e6b3437ff1810c3420884ffbe13a8826c2830d7bec2c586c95a33f1572146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089a4e1a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a2e39ff70510a5fe848d032240bdbb92c73dd1bd5a61ebc72099b76d3d934573a5a54196a04e87e010964864d34e2a2a42ada217c5c9be210adc1cd0b2091c32fb9525fdabfb247fd7660466001a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10011, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189b4e220c08a2e39ff70510a5fe848d032a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c322055b77f9a764871721521ff6aeb920fe01bc440446712b483866cd4d09944839e422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20b4cf9c1ec8e800ee5e201678c25580124ea8ae1e3dc7c4128d67b92ac876522f72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089b4e1a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a7e39ff70510ecbbe6b903224048b1c9b7b32bdc0ecc11cb491dcd23f621bb799969221dae82e28645d229310eac1e3cde1aae6056b395d75b4d3adc96190c3cc40dc1ecaa2e431f4d5b78620b1a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10010)
		param.Headers = append(param.Headers, header10011)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), new(types.Transaction), native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10011))
	}
	{
		header10012, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189c4e220c08a7e39ff70510ecbbe6b9032a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b32202496df827069967ea78dd93d07e6389e95abadd81cb5bd53cb9048b73a7ae0f2422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a206a1f58abdc0c358ed42023b6da978c6baa7c748f2534f42cec212c5ddeddd16172146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b601089c4e1a480a20744cf79eec130b744c9325ee44eb74661cfbe090989ee27af6a6856750ff41751224080112204a684ab601bb25ede19c18622513ecd95637dff4ef6270a1cec154e35a939a212267080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0b08ade39ff70510e9cad70b2240d448c0de45611b935f1f4d37ca33cafa174c9eebdc074febdf70eaac9fc384bf669369c46ed279b18f5bd7a73999d4ae46c382204947f019fa3fca1ea9d0dc021a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10011, _ := hex.DecodeString("0aa8020a02080a120774657374696e67189b4e220c08a2e39ff70510a5fe848d032a480a2064852da2991661fbd8d4f5e6789450b57662248ca4b996f640b327564c7c84601224080112209537acb3f8eebab33b949353ca450207614152c7df471b3aff1ffb683f78939c322055b77f9a764871721521ff6aeb920fe01bc440446712b483866cd4d09944839e422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a20b4cf9c1ec8e800ee5e201678c25580124ea8ae1e3dc7c4128d67b92ac876522f72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b701089b4e1a480a20dd5ee8ddd3d21d365aa11153d8e83367140de92e4ccfabd999e97f093b83d388122408011220d3ee12d83dbba28047922ee735604065bbf18286ab770262327e92aed9d7662b2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c08a7e39ff70510ecbbe6b903224048b1c9b7b32bdc0ecc11cb491dcd23f621bb799969221dae82e28645d229310eac1e3cde1aae6056b395d75b4d3adc96190c3cc40dc1ecaa2e431f4d5b78620b1a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		header10008, _ := hex.DecodeString("0aa8020a02080a120774657374696e6718984e220c0893e39ff70510bc93c689022a480a2014a45a0bde0573b04e03e9510a9e5b892d900803daf0296b0304c7172b865e2f122408011220d6e71ceeb4cc3ba6e5969eb127f5978a520c65501fc7a05c0ec79207b832bb1c32202e5974d3a2e211545c3648129248bd17317c8b34e456c2fceebadcb83ee9d76b422058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883794a2058df3ad01815e689d296705e219563932f8edd3637c1cd8f4a785906ca8883795220048091bc7ddc283f77bfbf91d73c44da58c3df8a9cbc867405d8b7f3daada22f5a202e13fd7f9843a72be774ecfcd93e2dfc6ece88af940606398c7d7dfc1aadf7ba72146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812b70108984e1a480a20785bf8c3118aca98f1afa8138efed02c9e56056b4081b0aa99199f9e40590f1112240801122017d6baffe9414cafd8c348b42e13899a561faf72d49d9493b23711d43ed480dd2268080212146ff75a0ce1ed3596eb34a107bcfc1bebd1ea94781a0c0898e39ff70510a1bb83be02224064ef0539008ed04defb72325818b79aca8a1aac9ec68e82c32177d6b28f37ee6d4b45576877da5ce734fb3f031f45156dfed366401024f88596dbfa9c47bd1041a3f0a146ff75a0ce1ed3596eb34a107bcfc1bebd1ea947812251624de6420760145874ef07a40698eea7afdf7d89719c76c96a5517ac2cf1162bb2e0a70a21864")
		param := new(scom.SyncBlockHeaderParam)
		param.ChainID = 5
		param.Address = acct.Address
		param.Headers = append(param.Headers, header10012)
		param.Headers = append(param.Headers, header10011)
		param.Headers = append(param.Headers, header10008)
		sink := common.NewZeroCopySink(nil)
		param.Serialization(sink)

		native = NewNative(sink.Bytes(), nil, native.GetCacheDB())
		err := cosmosHandler.SyncBlockHeader(native)
		if err != nil {
			fmt.Printf("err: %s", err.Error())
		}
		assert.Equal(t, SUCCESS, typeOfError(err))

		//height, _ := getCurrentHeight(native, param.ChainID)
		//assert.Equal(t, uint64(height), uint64(10012))
	}
}
