package module

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"../errors"
)

// 默认的组件序列号生成器
var DefaultSNGen = NewSNGenertor(1, 0)

// 组件ID的模板
var midTemplate = "%s%d|%s"

// 组件ID
type MID string

// 根据给定参数生成组件ID
func GenMID(mType Type, sn uint64, mAddr net.Addr) (MID, error) {
	if !LegalType(mType) {
		errMsg := fmt.Sprintf("非法模块类型: %s", mType)
		return "", errors.NewIllegalParameterError(errMsg)
	}
	letter := legalTypeLetterMap[mType]
	var midStr string
	if mAddr == nil {
		midStr = fmt.Sprintf(midTemplate, letter, sn, "")
		midStr = midStr[:len(midStr)-1]
	} else {
		midStr = fmt.Sprintf(midTemplate, letter, sn, mAddr.String())
	}
	return MID(midStr), nil
}

// 用于判断给定的组件ID是否合法
func LegalMID(mid MID) bool {
	if _, err := SplitMID(mid); err == nil {
		return true
	}
	return false
}

// 用于分解组件ID
// 第一个结果值表示分解是否成功
// 若分解成功，则第二个结果值长度为3
// 并依次包含组件类型字母，序列号和组件网络地址（如果有的话）
func SplitMID(mid MID) ([]string, error) {
	var ok bool
	var letter string
	var snStr string
	var addr string
	midStr := string(mid)
	if len(midStr) <= 1 {
		return nil, errors.NewIllegalParameterError("组件MID长度不正确")
	}
	letter = midStr[:1]
	if _, ok = legalLetterTypeMap[letter]; !ok {
		return nil, errors.NewIllegalParameterError(fmt.Sprintf("组件类型不正确: %s", letter))
	}
	snAndAddr := midStr[1:]
	index := strings.LastIndex(snAndAddr, "|")
	if index < 0 {
		snStr = snAndAddr
		if !legalSN(snStr) {
			return nil, errors.NewIllegalParameterError(fmt.Sprintf("序列号不合法: %s", snStr))
		}
	} else {
		snStr = snAndAddr[:index]
		if !legalSN(snStr) {
			if !legalSN(snStr) {
				return nil, errors.NewIllegalParameterError(fmt.Sprintf("序列号不合法: %s", snStr))
			}
		}
		addr = snAndAddr[index+1:]
		index = strings.LastIndex(addr, ":")
		if index <= 0 {
			return nil, errors.NewIllegalParameterError(fmt.Sprintf("地址不合法: %s", addr))
		}
		ipStr := addr[:index]
		if ip := net.ParseIP(ipStr); ip == nil {
			return nil, errors.NewIllegalParameterError(fmt.Sprintf("IP不合法: %s", ipStr))
		}
		portStr := addr[index+1:]
		if _, err := strconv.ParseUint(portStr, 10, 64); err != nil {
			return nil, errors.NewIllegalParameterError(fmt.Sprintf("端口不合法: %s", portStr))
		}
	}
	return []string{letter, snStr, addr}, nil
}

// 用于判断序列号的合法性
func legalSN(snStr string) bool {
	_, err := strconv.ParseUint(snStr, 10, 64)
	if err != nil {
		return false
	}
	return true
}
