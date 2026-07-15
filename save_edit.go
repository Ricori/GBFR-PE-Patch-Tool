package main

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

// save_edit.go — 通过存档文件（离线）修改角色使用次数与副本(任务)完成次数。
// 复用 sigil_store.go 的 SaveData 原地改字节 + FixChecksums + Write 流程，
// 与游戏版本、进程内存偏移无关。

// SetUint32At 写入向量单元中第 i 个 uint32 元素（用于副本次数这类数组单元）。
func (e *unitEntry) SetUint32At(i int, v uint32) {
	off := e.ValueOff + i*4
	if off < 0 || off+4 > len(e.data) {
		return
	}
	binary.LittleEndian.PutUint32(e.data[off:], v)
}

// defaultModifiedPath 在原文件名后加 _modified，作为默认输出路径。
func defaultModifiedPath(input string) string {
	ext := filepath.Ext(input)
	return strings.TrimSuffix(input, ext) + "_modified" + ext
}

// ── 角色使用次数 ──

// CharaCountRow 是可编辑的单个角色次数行。Slot 为角色槽位（UnitID = 10000 + Slot）。
type CharaCountRow struct {
	Slot  int    `json:"slot"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
}

func editCharacterNames(newSave bool) []string {
	oldCharacterNames := []string{
		"古兰", "姬塔", "卡塔莉娜", "拉卡姆", "伊欧", "欧根", "", "萝赛塔", "冈达葛萨", "菲莉",
		"兰斯洛特", "巴恩", "珀西瓦尔", "", "齐格飞", "夏洛特", "索恩", "尤达拉哈", "娜露梅", "伽兰查",
		"塞达", "伊德", "巴萨拉卡", "", "卡莉奥丝特罗", "", "", "圣德芬", "希耶提", "",
		"", "", "", "", "", "", "菲迪埃尔", "贝阿朵丽丝", "玛琪拉菲菈", "尤斯提斯",
		"芙劳", "", "", "", "", "", "", "", "", "",
	}
	newCharacterNames := []string{
		"古兰", "姬塔", "菲迪埃尔", "卡塔莉娜", "拉卡姆", "伊欧", "欧根", "", "萝赛塔", "冈达葛萨",
		"菲莉", "兰斯洛特", "贝阿朵丽丝", "巴恩", "珀西瓦尔", "", "齐格飞", "夏洛特", "索恩", "尤达拉哈",
		"娜露梅", "伽兰查", "塞达", "伊德", "巴萨拉卡", "", "卡莉奥丝特罗", "", "", "圣德芬",
		"希耶提", "玛琪拉菲菈", "尤斯提斯", "", "芙劳", "", "", "", "", "",
	}
	if newSave {
		return newCharacterNames
	}
	return oldCharacterNames
}

// GetCharacterStatsEditable 读取角色次数（带槽位，供编辑用）。
func (a *App) GetCharacterStatsEditable(path string, newSave bool) ([]CharaCountRow, error) {
	save, err := LoadSaveFile(path)
	if err != nil {
		return nil, err
	}
	if save.SlotData == nil {
		return nil, fmt.Errorf("存档SlotData为空")
	}

	const firstCharacterSlot uint32 = 10000
	counts := make(map[uint32]uint32, 41)
	for _, unit := range save.SlotData.UIntTable {
		if unit.IDType == SaveID_CharacterQuestUse && len(unit.ValueData) > 0 &&
			unit.UnitID >= firstCharacterSlot && unit.UnitID < firstCharacterSlot+41 {
			counts[unit.UnitID-firstCharacterSlot] = unit.ValueData[0]
		}
	}

	names := editCharacterNames(newSave)
	rows := make([]CharaCountRow, 0, len(names))
	for slot, name := range names {
		if name == "" {
			continue
		}
		rows = append(rows, CharaCountRow{Slot: slot, Name: name, Count: counts[uint32(slot)]})
	}
	return rows, nil
}

// ApplyCharacterStats 把逐个角色的次数写入新存档，返回输出文件路径。
func (a *App) ApplyCharacterStats(inputPath, outputPath string, edits []CharaCountRow) (string, error) {
	if inputPath == "" {
		return "", fmt.Errorf("未指定存档路径")
	}
	if outputPath == "" {
		outputPath = defaultModifiedPath(inputPath)
	}

	sd, err := LoadSave(inputPath)
	if err != nil {
		return "", err
	}

	const firstCharacterSlot = 10000
	applied := 0
	for _, e := range edits {
		// 该角色在存档中若无对应 unit（未获得），patchUint 返回错误，跳过即可。
		if err := sd.patchUint(SaveID_CharacterQuestUse, uint32(firstCharacterSlot+e.Slot), e.Count); err != nil {
			continue
		}
		applied++
	}
	if applied == 0 {
		return "", fmt.Errorf("没有可写入的角色次数（存档中未找到对应角色）")
	}

	if err := sd.FixChecksums(); err != nil {
		return "", fmt.Errorf("重算校验和失败: %w", err)
	}
	if err := sd.Write(outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
}

// ── 点赞数 (Commendations) ──

// MaxCommendations 是游戏内部对点赞数的上限（exe 中 clamp 到 999999）。
const MaxCommendations = 999999

// ApplyCommendations 把点赞数写入新存档，返回输出文件路径。
// 取代原先的 PE 补丁方案：补丁改的是运行时 clamp、要被点赞一次才落存档，
// 且随游戏版本更新失效；直接写存档等价且与版本无关。
func (a *App) ApplyCommendations(inputPath, outputPath string, value int32) (string, error) {
	if inputPath == "" {
		return "", fmt.Errorf("未指定存档路径")
	}
	if value < 0 || value > MaxCommendations {
		return "", fmt.Errorf("点赞数需在 0 ~ %d 之间", MaxCommendations)
	}
	if outputPath == "" {
		outputPath = defaultModifiedPath(inputPath)
	}

	sd, err := LoadSave(inputPath)
	if err != nil {
		return "", err
	}

	if err := sd.patchInt(SaveID_Commendations, 0, int(value)); err != nil {
		return "", fmt.Errorf("找不到点赞数据 (IDType=%d): %w", SaveID_Commendations, err)
	}

	if err := sd.FixChecksums(); err != nil {
		return "", fmt.Errorf("重算校验和失败: %w", err)
	}
	if err := sd.Write(outputPath); err != nil {
		return "", err
	}

	// 回读验证：写入走的是 LoadSave 的字节扫描定位，这里用 LoadSaveFile 的
	// FlatBuffer 解析回读，两套解析器交叉校验，避免写到错误的偏移而无人察觉。
	verify, err := LoadSaveFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("回读验证失败: %w", err)
	}
	if verify.SlotData == nil {
		return "", fmt.Errorf("回读验证失败: 输出存档 SlotData 为空")
	}
	unit := verify.SlotData.GetIntUnit(SaveID_Commendations)
	if unit == nil || len(unit.ValueData) == 0 {
		return "", fmt.Errorf("回读验证失败: 输出存档中找不到点赞数据")
	}
	if unit.ValueData[0] != value {
		return "", fmt.Errorf("回读验证失败: 期望 %d, 实际 %d", value, unit.ValueData[0])
	}

	return outputPath, nil
}

// ── 副本(任务)完成次数 ──

// QuestCountRow 是可编辑的单个副本次数行。Index 为其在完成次数向量中的位置。
type QuestCountRow struct {
	Index       int    `json:"index"`
	QuestID     uint32 `json:"questId"`
	QuestName   string `json:"questName"`
	QuestNameCN string `json:"questNameCn"`
	Clears      uint32 `json:"clears"`
}

// GetQuestsEditable 读取副本列表（带向量索引，供编辑用）。
func (a *App) GetQuestsEditable(path string) ([]QuestCountRow, error) {
	save, err := LoadSaveFile(path)
	if err != nil {
		return nil, err
	}
	if save.SlotData == nil {
		return nil, fmt.Errorf("存档SlotData为空")
	}

	qIDs := save.SlotData.GetUIntUnit(SaveID_QuestIDs)
	qCounts := save.SlotData.GetUIntUnit(SaveID_QuestCompleteCount)
	if qIDs == nil || qCounts == nil {
		return nil, nil
	}

	var rows []QuestCountRow
	for i := 0; i < len(qIDs.ValueData); i++ {
		if qIDs.ValueData[i] == 0 {
			continue
		}
		count := uint32(0)
		if i < len(qCounts.ValueData) {
			count = qCounts.ValueData[i]
		}
		rows = append(rows, QuestCountRow{
			Index:       i,
			QuestID:     storedToQuestID(qIDs.ValueData[i]),
			QuestName:   questIDToName(qIDs.ValueData[i]),
			QuestNameCN: questIDToNameCN(qIDs.ValueData[i]),
			Clears:      count,
		})
	}
	return rows, nil
}

// ApplyQuestCounts 把逐个副本的完成次数写入新存档，返回输出文件路径。
func (a *App) ApplyQuestCounts(inputPath, outputPath string, edits []QuestCountRow) (string, error) {
	if inputPath == "" {
		return "", fmt.Errorf("未指定存档路径")
	}
	if outputPath == "" {
		outputPath = defaultModifiedPath(inputPath)
	}

	sd, err := LoadSave(inputPath)
	if err != nil {
		return "", err
	}

	entry, ok := sd.findUnit(SaveID_QuestCompleteCount, 0)
	if !ok {
		return "", fmt.Errorf("找不到副本完成次数数据 (IDType=%d)", SaveID_QuestCompleteCount)
	}

	applied := 0
	for _, e := range edits {
		if e.Index < 0 || e.Index >= entry.ValueCnt {
			continue
		}
		entry.SetUint32At(e.Index, e.Clears)
		applied++
	}
	if applied == 0 {
		return "", fmt.Errorf("没有可写入的副本次数")
	}

	if err := sd.FixChecksums(); err != nil {
		return "", fmt.Errorf("重算校验和失败: %w", err)
	}
	if err := sd.Write(outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
}
