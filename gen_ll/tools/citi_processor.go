package tools

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// CitiEntry 表示一个编码条目
type CitiEntry struct {
	Text     string // 字或词
	Code     string // 编码
	Freq     int64  // 词频
	Source   string // 来源文件标识
}

// ReadCitiFile 读取编码文件并解析为CitiEntry列表
func ReadCitiFile(filepath string, source string) ([]*CitiEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件 %s: %w", filepath, err)
	}
	defer file.Close()

	var entries []*CitiEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}

		entry := &CitiEntry{
			Text:   fields[0],
			Code:   fields[1],
			Source: source,
		}

		// 如果有第三列，解析词频
		if len(fields) >= 3 {
			freq, err := strconv.ParseInt(fields[2], 10, 64)
			if err == nil {
				entry.Freq = freq
			}
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件 %s 时出错: %w", filepath, err)
	}

	return entries, nil
}

// SortByFreq 按词频降序排序
func SortByFreq(entries []*CitiEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Freq > entries[j].Freq
	})
}

// WriteCitiFile 将CitiEntry列表写入文件
func WriteCitiFile(filepath string, entries []*CitiEntry) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("无法创建文件 %s: %w", filepath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		line := fmt.Sprintf("%s\t%s\t%d\n", entry.Text, entry.Code, entry.Freq)
		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("写入文件 %s 时出错: %w", filepath, err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新文件 %s 时出错: %w", filepath, err)
	}

	return nil
}

// ProcessCitiFiles 处理四个编码文件：按词频重排
func ProcessCitiFiles(charsSimpFile, charsFullFile, wordsSimpFile, wordsFullFile string) error {
	// 读取四个文件
	charsSimpEntries, err := ReadCitiFile(charsSimpFile, "chars_simp")
	if err != nil {
		return fmt.Errorf("读取单字简码文件失败: %w", err)
	}

	charsFullEntries, err := ReadCitiFile(charsFullFile, "chars_full")
	if err != nil {
		return fmt.Errorf("读取单字全码文件失败: %w", err)
	}

	wordsSimpEntries, err := ReadCitiFile(wordsSimpFile, "words_simp")
	if err != nil {
		return fmt.Errorf("读取多字词简码文件失败: %w", err)
	}

	wordsFullEntries, err := ReadCitiFile(wordsFullFile, "words_full")
	if err != nil {
		return fmt.Errorf("读取多字词全码文件失败: %w", err)
	}

	// 按词频重排
	SortByFreq(charsSimpEntries)
	SortByFreq(charsFullEntries)
	SortByFreq(wordsSimpEntries)
	SortByFreq(wordsFullEntries)

	// 写回原文件
	if err := WriteCitiFile(charsSimpFile, charsSimpEntries); err != nil {
		return fmt.Errorf("写入单字简码文件失败: %w", err)
	}

	if err := WriteCitiFile(charsFullFile, charsFullEntries); err != nil {
		return fmt.Errorf("写入单字全码文件失败: %w", err)
	}

	if err := WriteCitiFile(wordsSimpFile, wordsSimpEntries); err != nil {
		return fmt.Errorf("写入多字词简码文件失败: %w", err)
	}

	if err := WriteCitiFile(wordsFullFile, wordsFullEntries); err != nil {
		return fmt.Errorf("写入多字词全码文件失败: %w", err)
	}

	return nil
}

// CombineCitiFiles 将四个文件按照指定顺序拼接在一起
func CombineCitiFiles(charsSimpFile, charsFullFile, wordsSimpFile, wordsFullFile string) ([]*CitiEntry, error) {
	var allEntries []*CitiEntry

	// 按照指定顺序读取四个文件
	files := []struct {
		path   string
		source string
	}{
		{charsSimpFile, "chars_simp"},
		{charsFullFile, "chars_full"},
		{wordsSimpFile, "words_simp"},
		{wordsFullFile, "words_full"},
	}

	for _, file := range files {
		entries, err := ReadCitiFile(file.path, file.source)
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %w", file.path, err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// CombineAllCitiFiles 按照指定顺序组合所有文件：ll_citi_pre + 四个编码文件
func CombineAllCitiFiles(citiPreFile, charsSimpFile, charsFullFile, wordsSimpFile, wordsFullFile string) ([]*CitiEntry, error) {
	var allEntries []*CitiEntry

	// 1. 首先读取现有的ll_citi_pre.txt内容
	existingEntries, err := ReadCitiFile(citiPreFile, "citi_pre")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取现有文件失败: %w", err)
	}
	allEntries = append(allEntries, existingEntries...)

	// 2. 然后按照指定顺序读取四个编码文件
	files := []struct {
		path   string
		source string
	}{
		{charsSimpFile, "chars_simp"},
		{charsFullFile, "chars_full"},
		{wordsSimpFile, "words_simp"},
		{wordsFullFile, "words_full"},
	}

	for _, file := range files {
		entries, err := ReadCitiFile(file.path, file.source)
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %w", file.path, err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// AppendToCitiPre 将合并的条目追加到ll_citi_pre.txt
func AppendToCitiPre(entries []*CitiEntry, citiPreFile string) error {
	// 读取现有的ll_citi_pre.txt内容
	existingEntries, err := ReadCitiFile(citiPreFile, "existing")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取现有文件失败: %w", err)
	}

	// 合并现有条目和新条目
	allEntries := append(existingEntries, entries...)

	// 写入文件
	file, err := os.Create(citiPreFile)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range allEntries {
		line := fmt.Sprintf("%s\t%s\n", entry.Text, entry.Code)
		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新文件失败: %w", err)
	}

	return nil
}

// CreateGendaCiti 创建genda_citi.txt并删除词频
func CreateGendaCiti(entries []*CitiEntry, gendaCitiFile string) error {
	file, err := os.Create(gendaCitiFile)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		line := fmt.Sprintf("%s\t%s\n", entry.Text, entry.Code)
		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新文件失败: %w", err)
	}

	return nil
}

// AddCandidateCodes 为重复编码添加候选码，保持原始文件顺序
func AddCandidateCodes(entries []*CitiEntry) []*CitiEntry {
	// 按编码分组，但记录每个条目的原始位置
	type entryWithIndex struct {
		entry *CitiEntry
		index int
	}
	codeGroups := make(map[string][]*entryWithIndex)
	
	for i, entry := range entries {
		codeGroups[entry.Code] = append(codeGroups[entry.Code], &entryWithIndex{entry, i})
	}

	// 创建结果数组，保持原始顺序
	result := make([]*CitiEntry, len(entries))
	candidateSuffixes := []string{"_", "e", "i", "[", "2", "3", "7", "8", "9", "0"}

	// 处理每个编码的重码情况
	for code, group := range codeGroups {
		if len(group) == 1 {
			// 没有重码，直接使用原编码
			result[group[0].index] = group[0].entry
			continue
		}

		// 有重码，按词频排序（保持词频排序）
		sort.Slice(group, func(i, j int) bool {
			return group[i].entry.Freq > group[j].entry.Freq
		})

		// 为每个候选添加后缀，保持原始位置
		for i, ew := range group {
			var newCode string
			if i < 10 {
				// 前10个候选使用单字符后缀
				newCode = code + candidateSuffixes[i]
			} else {
				// 第11个及以后的候选使用翻页格式
				page := (i - 10) / 10
				posInPage := (i - 10) % 10
				// 第1页：=_, =e, =i, =[, =2, =3, =7, =8, =9, =0
				// 第2页：==_, ==e, ==i, ==[, ==2, ==3, ==7, ==8, ==9, ==0
				// 第3页：===_, ===e, 以此类推...
				equals := strings.Repeat("=", page+1)
				newCode = fmt.Sprintf("%s%s%s", code, equals, candidateSuffixes[posInPage])
			}

			newEntry := &CitiEntry{
				Text:   ew.entry.Text,
				Code:   newCode,
				Freq:   ew.entry.Freq,
				Source: ew.entry.Source,
			}
			result[ew.index] = newEntry
		}
	}

	// 移除可能为nil的条目（理论上不应该有）
	finalResult := make([]*CitiEntry, 0, len(entries))
	for _, entry := range result {
		if entry != nil {
			finalResult = append(finalResult, entry)
		}
	}

	return finalResult
}

// AddCandidateCodesForDazhu 为重复编码添加候选码（大竹专用版本，不使用后缀）
func AddCandidateCodesForDazhu(entries []*CitiEntry) []*CitiEntry {
	// 按编码分组，但记录每个条目的原始位置
	type entryWithIndex struct {
		entry *CitiEntry
		index int
	}
	codeGroups := make(map[string][]*entryWithIndex)
	
	for i, entry := range entries {
		codeGroups[entry.Code] = append(codeGroups[entry.Code], &entryWithIndex{entry, i})
	}

	// 创建结果数组，保持原始顺序
	result := make([]*CitiEntry, len(entries))

	// 处理每个编码的重码情况
	for code, group := range codeGroups {
		if len(group) == 1 {
			// 没有重码，直接使用原编码
			result[group[0].index] = group[0].entry
			continue
		}

		// 有重码，按词频排序（保持词频排序）
		sort.Slice(group, func(i, j int) bool {
			return group[i].entry.Freq > group[j].entry.Freq
		})

		// 为每个候选使用原编码，不添加任何后缀
		// 避免生成类似 _ei[237890 的后缀
		for _, ew := range group {
			newEntry := &CitiEntry{
				Text:   ew.entry.Text,
				Code:   code, // 直接使用原编码，不添加后缀
				Freq:   ew.entry.Freq,
				Source: ew.entry.Source,
			}
			result[ew.index] = newEntry
		}
	}

	// 移除可能为nil的条目（理论上不应该有）
	finalResult := make([]*CitiEntry, 0, len(entries))
	for _, entry := range result {
		if entry != nil {
			finalResult = append(finalResult, entry)
		}
	}

	return finalResult
}

// ProcessCitiFilesComplete 完整的citi文件处理流程
func ProcessCitiFilesComplete(charsSimpFile, charsFullFile, wordsSimpFile, wordsFullFile, citiPreFile, gendaCitiFile string) error {
	// 按照指定顺序分别处理每个来源，保持各自原始排序
	var allEntries []*CitiEntry

	// 1. 首先处理ll_citi_pre.txt - 不进行重码处理，保持原有顺序
	citiPreEntries, err := ReadCitiFile(citiPreFile, "citi_pre")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取ll_citi_pre.txt失败: %w", err)
	}
	// ll_citi_pre.txt已经包含候选编码补码，直接使用
	allEntries = append(allEntries, citiPreEntries...)

	// 2. 然后处理code_chars_simp.txt - 不需要运用补码规则，直接使用
	charsSimpEntries, err := ReadCitiFile(charsSimpFile, "chars_simp")
	if err != nil {
		return fmt.Errorf("读取code_chars_simp.txt失败: %w", err)
	}
	allEntries = append(allEntries, charsSimpEntries...)

	// 3. 接着处理code_chars_full.txt - 需要运用补码规则
	charsFullEntries, err := ReadCitiFile(charsFullFile, "chars_full")
	if err != nil {
		return fmt.Errorf("读取code_chars_full.txt失败: %w", err)
	}
	charsFullWithCandidates := AddCandidateCodes(charsFullEntries)
	allEntries = append(allEntries, charsFullWithCandidates...)

	// 4. 然后处理code_words_simp.txt - 需要运用补码规则
	wordsSimpEntries, err := ReadCitiFile(wordsSimpFile, "words_simp")
	if err != nil {
		return fmt.Errorf("读取code_words_simp.txt失败: %w", err)
	}
	wordsSimpWithCandidates := AddCandidateCodes(wordsSimpEntries)
	allEntries = append(allEntries, wordsSimpWithCandidates...)

	// 5. 最后处理code_words_full.txt - 需要运用补码规则
	wordsFullEntries, err := ReadCitiFile(wordsFullFile, "words_full")
	if err != nil {
		return fmt.Errorf("读取code_words_full.txt失败: %w", err)
	}
	wordsFullWithCandidates := AddCandidateCodes(wordsFullEntries)
	allEntries = append(allEntries, wordsFullWithCandidates...)

	// 创建genda_citi.txt并删除词频
	if err := CreateGendaCiti(allEntries, gendaCitiFile); err != nil {
		return fmt.Errorf("创建genda_citi.txt失败: %w", err)
	}

	return nil
}

// CreateDazhuCode 根据genda_citi.txt生成dazhu_code.txt，格式为"编码\t字词"
func CreateDazhuCode(gendaCitiFile, dazhuCodeFile string, maxSizeMB int) error {
	// 读取genda_citi.txt文件
	entries, err := ReadCitiFile(gendaCitiFile, "genda_citi")
	if err != nil {
		return fmt.Errorf("读取genda_citi.txt失败: %w", err)
	}

	// 创建输出文件
	file, err := os.Create(dazhuCodeFile)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	maxSizeBytes := maxSizeMB * 1024 * 1024
	currentSize := 0

	// 按"编码\t字词"格式写入，并控制文件大小
	for _, entry := range entries {
		line := fmt.Sprintf("%s\t%s\n", entry.Code, entry.Text)
		lineSize := len([]byte(line))
		
		// 检查是否超过最大文件大小
		if currentSize+lineSize > maxSizeBytes {
			break
		}
		
		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
		currentSize += lineSize
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新文件失败: %w", err)
	}

	return nil
}
// ProcessCitiFilesCompleteForDazhu 完整的citi文件处理流程（大竹专用版本，不使用后缀）
func ProcessCitiFilesCompleteForDazhu(charsSimpFile, charsFullFile, wordsSimpFile, wordsFullFile, citiPreFile, gendaCitiFile string) error {
	// 按照指定顺序分别处理每个来源，保持各自原始排序
	var allEntries []*CitiEntry

	// 1. 首先处理ll_citi_pre.txt - 不进行重码处理，保持原有顺序
	citiPreEntries, err := ReadCitiFile(citiPreFile, "citi_pre")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取ll_citi_pre.txt失败: %w", err)
	}
	// ll_citi_pre.txt已经包含候选编码补码，直接使用
	allEntries = append(allEntries, citiPreEntries...)

	// 2. 然后处理code_chars_simp.txt - 不需要运用补码规则，直接使用
	charsSimpEntries, err := ReadCitiFile(charsSimpFile, "chars_simp")
	if err != nil {
		return fmt.Errorf("读取code_chars_simp.txt失败: %w", err)
	}
	allEntries = append(allEntries, charsSimpEntries...)

	// 3. 接着处理code_chars_full.txt - 需要运用补码规则（大竹专用版本）
	charsFullEntries, err := ReadCitiFile(charsFullFile, "chars_full")
	if err != nil {
		return fmt.Errorf("读取code_chars_full.txt失败: %w", err)
	}
	charsFullWithCandidates := AddCandidateCodesForDazhu(charsFullEntries)
	allEntries = append(allEntries, charsFullWithCandidates...)

	// 4. 然后处理code_words_simp.txt - 需要运用补码规则（大竹专用版本）
	wordsSimpEntries, err := ReadCitiFile(wordsSimpFile, "words_simp")
	if err != nil {
		return fmt.Errorf("读取code_words_simp.txt失败: %w", err)
	}
	wordsSimpWithCandidates := AddCandidateCodesForDazhu(wordsSimpEntries)
	allEntries = append(allEntries, wordsSimpWithCandidates...)

	// 5. 最后处理code_words_full.txt - 需要运用补码规则（大竹专用版本）
	wordsFullEntries, err := ReadCitiFile(wordsFullFile, "words_full")
	if err != nil {
		return fmt.Errorf("读取code_words_full.txt失败: %w", err)
	}
	wordsFullWithCandidates := AddCandidateCodesForDazhu(wordsFullEntries)
	allEntries = append(allEntries, wordsFullWithCandidates...)

	// 创建genda_citi.txt并删除词频
	if err := CreateGendaCiti(allEntries, gendaCitiFile); err != nil {
		return fmt.Errorf("创建genda_citi.txt失败: %w", err)
	}

	return nil
}