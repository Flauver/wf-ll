package tools

import (
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gen_ll/types"
)

const fallBackFreq = 100


// BuildFullCodeMetaList 构造字符四码全码编码列表
func BuildFullCodeMetaList(table map[string][]*types.Division, mappings map[string]string, freqSet map[string]int64) (charMetaList []*types.CharMeta) {
	// 预分配足够大的切片
	charMetaList = make([]*types.CharMeta, 0, len(table))
	
	// 并发处理以提高性能
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// 将字符表分块并行处理
	chars := make([]string, 0, len(table))
	for char := range table {
		chars = append(chars, char)
	}
	
	// 决定并发数量，根据CPU核心数自动调整
	concurrency := runtime.NumCPU()
	batchSize := (len(chars) + concurrency - 1) / concurrency
	
	for i := 0; i < concurrency; i++ {
		start := i * batchSize
		end := (i + 1) * batchSize
		if end > len(chars) {
			end = len(chars)
		}
		
		if start >= end {
			continue
		}
		
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			localCharMetaList := make([]*types.CharMeta, 0, end-start)
			
			// 处理当前批次的字符
			for i := start; i < end; i++ {
				char := chars[i]
				divs := table[char]
				
				// 遍历字符的所有拆分表
				for i, div := range divs {
					full, code := calcFullCodeByDiv(div.Divs, mappings)
					charMeta := types.CharMeta{
						Char:     char,
						Full:     full,
						Code:     code,
						Freq:     freqSet[char],
						MDiv:     i == 0,
						Division: div, // 绑定对应的拆分信息
					}
					
					localCharMetaList = append(localCharMetaList, &charMeta)
				}
			}
			
			// 合并本地结果到全局列表
			mutex.Lock()
			charMetaList = append(charMetaList, localCharMetaList...)
			mutex.Unlock()
		}(start, end)
	}
	
	// 等待所有协程完成
	wg.Wait()
	
	// 排序结果
	sortCharMetaByCode(charMetaList)
	return
}


func sortCharMetaByCode(charMetaList []*types.CharMeta) {
	// 按编码升序排列，对于相同编码的重码按词频降序排列
	sort.Slice(charMetaList, func(i, j int) bool {
		a, b := charMetaList[i], charMetaList[j]
		
		// 首先按编码升序排列
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		
		// 编码相同，按词频降序排列
		if a.Freq != b.Freq {
			return a.Freq > b.Freq
		}
		
		// 编码和词频都相同，按字符Unicode编码升序排列
		return a.Char < b.Char
	})
}


func calcFullCodeByDiv(div []string, mappings map[string]string) (full string, code string) {
	// 遍历处理每个部件，生成全码
	for i, comp := range div {
		compCode := mappings[comp]
		if len(compCode) == 0 {
			continue
		}
		// 在各部件编码之间添加"_"分隔符
		if i > 0 {
			full += "_"
		}
		full += compCode
	}
	
	// 根据拆分部件数量生成编码
	if len(div) == 1 {
		// 单根字处理
		compCode := mappings[div[0]]
		if len(compCode) == 0 {
			return "", ""
		}
		
		// 第一码：取部件大码（编码第一位）
		code = compCode[:1]
		
		// 第二码：取部件中码
		if len(compCode) >= 2 {
			code += compCode[1:2]
		} else {
			// 如果只有1码，重复大码
			code += compCode[:1]
		}
		
		// 第三码：取部件中码（重复第二码）
		if len(compCode) >= 2 {
			code += compCode[1:2]
		} else {
			// 如果只有1码，重复大码
			code += compCode[:1]
		}
		
		// 第四码：取部件小码
		if len(compCode) >= 3 {
			code += compCode[2:3]
		} else if len(compCode) == 2 {
			// 如果只有双编码，取中码
			code += compCode[1:2]
		} else {
			// 如果只有1码，重复大码
			code += compCode[:1]
		}
		
	} else if len(div) == 2 {
		// 双根字处理
		firstCompCode := mappings[div[0]]
		secondCompCode := mappings[div[1]]
		
		if len(firstCompCode) == 0 || len(secondCompCode) == 0 {
			return "", ""
		}
		
		// 第一码：第一部件大码
		code = firstCompCode[:1]
		
		// 第二码：第二部件大码
		code += secondCompCode[:1]
		
		// 第三码：第一部件中码
		if len(firstCompCode) >= 2 {
			code += firstCompCode[1:2]
		} else {
			// 如果只有1码，重复大码
			code += firstCompCode[:1]
		}
		
		// 第四码：第二部件小码
		if len(secondCompCode) >= 3 {
			code += secondCompCode[2:3]
		} else if len(secondCompCode) == 2 {
			// 如果只有双编码，取中码
			code += secondCompCode[1:2]
		} else {
			// 如果只有1码，重复大码
			code += secondCompCode[:1]
		}
		
	} else {
		// 三根字及以上多根字处理
		firstCompCode := mappings[div[0]]
		secondCompCode := mappings[div[1]]
		lastCompCode := mappings[div[len(div)-1]]
		
		if len(firstCompCode) == 0 || len(secondCompCode) == 0 || len(lastCompCode) == 0 {
			return "", ""
		}
		
		// 第一码：第一部件大码
		code = firstCompCode[:1]
		
		// 第二码：第二部件大码
		code += secondCompCode[:1]
		
		// 第三码：末部件大码
		code += lastCompCode[:1]
		
		// 第四码：末部件小码
		if len(lastCompCode) >= 3 {
			code += lastCompCode[2:3]
		} else if len(lastCompCode) == 2 {
			// 如果只有双编码，取中码
			code += lastCompCode[1:2]
		} else {
			// 如果只有1码，重复大码
			code += lastCompCode[:1]
		}
	}
	
	// 确保编码长度不超过4码
	if len(code) > 4 {
		code = code[:4]
	}
	
	code = strings.ToLower(code)
	return
}

// ParseLenCodeLimit 解析简码长度限制字符串
func ParseLenCodeLimit(limitStr string) (map[int]int, error) {
	limits := make(map[int]int)
	if limitStr == "" {
		return limits, nil
	}
	
	pairs := strings.Split(limitStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			continue
		}
		
		length, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		
		limit, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		
		limits[length] = limit
	}
	
	return limits, nil
}

// BuildSimpleCodeList 构建简码列表
func BuildSimpleCodeList(fullCodeList []*types.CharMeta, lenCodeLimit map[int]int, noSimplifyChars []string) []*types.CharMeta {
	// 按词频排序
	sortedList := make([]*types.CharMeta, len(fullCodeList))
	copy(sortedList, fullCodeList)
	sort.Slice(sortedList, func(i, j int) bool {
		return sortedList[i].Freq > sortedList[j].Freq
	})
	
	// 出简不出全 - 只保留成功简化的条目
	resultData := make([]*types.CharMeta, 0)
	usedCodes := make(map[string]bool)
	
	// 创建不出简字符的集合
	noSimplifySet := make(map[string]bool)
	for _, char := range noSimplifyChars {
		noSimplifySet[char] = true
	}
	
	for _, charMeta := range sortedList {
		word := charMeta.Char
		code := charMeta.Code
		freq := charMeta.Freq
		
		// 跳过不出简的字符
		if noSimplifySet[word] {
			continue
		}
		
		fullCodeLastChar := string(code[len(code)-1])
		var simplified string
		
		// 尝试生成简码
		for i := 0; i < len(code); i++ {
			limit := lenCodeLimit[i+1]
			if limit == 0 {
				continue
			}
			
			currentPrefix := code[:i+1]
			// 计算目标简码长度：1简和2简是前缀长度+1（因为加末码），3简及以上是前缀长度
			var targetLength int
			if i+1 <= 2 {
				targetLength = (i + 1) + 1
			} else {
				targetLength = i + 1
			}
			
			// 统计相同前缀的简码数量
			samePrefixCount := 0
			for _, res := range resultData {
				resCode := res.Code
				if len(resCode) == targetLength && strings.HasPrefix(resCode, currentPrefix) {
					samePrefixCount++
				}
			}
			
			if samePrefixCount >= limit {
				continue
			}
			
			// 生成候选简码
			var candidate string
			if i+1 <= 2 {
				candidate = currentPrefix + fullCodeLastChar
			} else {
				candidate = currentPrefix
			}
			
			if !usedCodes[candidate] {
				simplified = candidate
				usedCodes[simplified] = true
				break
			}
		}
		
		// 如果生成了简码且与全码不同，则添加到结果
		if simplified != "" && simplified != code {
			newCharMeta := &types.CharMeta{
				Char: word,
				Code: simplified,
				Freq: freq,
				Simp: true,
			}
			resultData = append(resultData, newCharMeta)
		}
	}
	
	return resultData
}


// BuildWordsFullCode 构建多字词全码
func BuildWordsFullCode(wordEntries []*types.WordEntry, charCodeMap map[string]string) []*types.WordCode {
	wordCodes := make([]*types.WordCode, 0, len(wordEntries))
	
	for _, entry := range wordEntries {
		word := entry.Word
		chars := []rune(word)
		
		// 根据词语长度应用不同的编码规则
		var code string
		switch len(chars) {
		case 2:
			// 二字词：取每个字编码的前2位，拼接成4位编码
			firstCode := charCodeMap[string(chars[0])]
			secondCode := charCodeMap[string(chars[1])]
			
			if len(firstCode) >= 2 && len(secondCode) >= 2 {
				code = firstCode[:2] + secondCode[:2]
			}
			
		case 3:
			// 三字词：前两个字各取编码的第1位，第三个字取编码的前2位
			firstCode := charCodeMap[string(chars[0])]
			secondCode := charCodeMap[string(chars[1])]
			thirdCode := charCodeMap[string(chars[2])]
			
			if len(firstCode) >= 1 && len(secondCode) >= 1 && len(thirdCode) >= 2 {
				code = firstCode[:1] + secondCode[:1] + thirdCode[:2]
			}
			
		default:
			// 四字及以上：取第一、二、三个字和最后一个字编码的第1位
			if len(chars) >= 4 {
				firstCode := charCodeMap[string(chars[0])]
				secondCode := charCodeMap[string(chars[1])]
				thirdCode := charCodeMap[string(chars[2])]
				lastCode := charCodeMap[string(chars[len(chars)-1])]
				
				if len(firstCode) >= 1 && len(secondCode) >= 1 && len(thirdCode) >= 1 && len(lastCode) >= 1 {
					code = firstCode[:1] + secondCode[:1] + thirdCode[:1] + lastCode[:1]
				}
			}
		}
		
		// 如果成功生成了编码，添加到结果列表
		if code != "" {
			wordCodes = append(wordCodes, &types.WordCode{
				Word:   word,
				Code:   code,
				Weight: entry.Weight,
			})
		}
	}
	
	return wordCodes
}

// CreateCharCodeMap 从字符元数据列表创建字符到编码的映射
func CreateCharCodeMap(charMetaList []*types.CharMeta) map[string]string {
	charCodeMap := make(map[string]string)
	
	for _, charMeta := range charMetaList {
		// 只使用主要拆分的编码
		if charMeta.MDiv {
			charCodeMap[charMeta.Char] = charMeta.Code
		}
	}
	
	return charCodeMap
}

// SortWordCodes 对多字词编码进行排序
// 排序规则：先按编码升序排列，编码相同时按权重降序排列
func SortWordCodes(wordCodes []*types.WordCode) {
	sort.Slice(wordCodes, func(i, j int) bool {
		a, b := wordCodes[i], wordCodes[j]
		
		// 首先按编码升序排列
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		
		// 编码相同，按权重降序排列
		// 将权重字符串转换为数值进行比较
		weightA := parseWeight(a.Weight)
		weightB := parseWeight(b.Weight)
		
		if weightA != weightB {
			return weightA > weightB
		}
		
		// 编码和权重都相同，按词语Unicode编码升序排列（保持稳定排序）
		return a.Word < b.Word
	})
}

// parseWeight 解析权重字符串为数值
// 如果权重为空或解析失败，返回默认值0
func parseWeight(weightStr string) int64 {
	if weightStr == "" {
		return 0
	}
	
	// 尝试解析为整数
	weight, err := strconv.ParseInt(weightStr, 10, 64)
	if err != nil {
		return 0
	}
	
	return weight
}

// BuildWordsSimpleCode 构建多字词简码
func BuildWordsSimpleCode(wordCodes []*types.WordCode, lenCodeLimit map[int]int) []*types.WordSimpleCode {
	// 按权重降序排序（权重高的优先分配简码）
	sortedWordCodes := make([]*types.WordCode, len(wordCodes))
	copy(sortedWordCodes, wordCodes)
	sort.Slice(sortedWordCodes, func(i, j int) bool {
		weightA := parseWeight(sortedWordCodes[i].Weight)
		weightB := parseWeight(sortedWordCodes[j].Weight)
		return weightA > weightB
	})

	// 初始化每个简码长度的计数器
	codeCounters := make(map[int]map[string]int)
	for length := 1; length <= 3; length++ {
		codeCounters[length] = make(map[string]int)
	}

	// 处理每个词
	resultData := make([]*types.WordSimpleCode, 0)
	for _, wordCode := range sortedWordCodes {
		word := wordCode.Word
		code := wordCode.Code
		weight := wordCode.Weight
		wordLength := len([]rune(word)) // 获取词的长度

		// 按照顺序尝试分配简码：先一简，再二简，最后三简
		var simplifiedCode string
		for codeLength := 1; codeLength <= 3; codeLength++ {
			// 检查该长度是否允许
			limit := lenCodeLimit[codeLength]
			if limit == 0 {
				continue
			}

			// 检查该长度的简码是否适用于当前词
			if codeLength == 2 && wordLength != 2 { // 二简只适用于二字词
				continue
			}
			if codeLength == 3 && wordLength != 3 { // 三简只适用于三字词
				continue
			}

			// 获取基础简码
			var baseCode string
			if codeLength == 2 && wordLength == 2 {
				// 二字词特殊规则：首码 + 第三个码
				if len(code) >= 3 {
					baseCode = code[:1] + code[2:3]
				} else {
					continue // 编码长度不足，跳过
				}
			} else {
				// 普通规则：取编码前codeLength位
				if len(code) >= codeLength {
					baseCode = code[:codeLength]
				} else {
					continue // 编码长度不足，跳过
				}
			}

			// 检查是否已达到该基础简码的限制
			currentCount := codeCounters[codeLength][baseCode]
			if currentCount < limit {
				// 创建新的简码条目
				simplifiedCode = baseCode

				resultData = append(resultData, &types.WordSimpleCode{
					Word:   word,
					Code:   simplifiedCode,
					Weight: weight,
				})
				codeCounters[codeLength][baseCode] = currentCount + 1
				break // 找到可用的简码后就不再尝试更长的简码
			}
		}
	}

	return resultData
}

// SortWordSimpleCodes 对多字词简码进行排序
// 排序规则：先按编码升序排列，编码相同时按权重降序排列
func SortWordSimpleCodes(wordSimpleCodes []*types.WordSimpleCode) {
	sort.Slice(wordSimpleCodes, func(i, j int) bool {
		a, b := wordSimpleCodes[i], wordSimpleCodes[j]

		// 首先按编码升序排列
		if a.Code != b.Code {
			return a.Code < b.Code
		}

		// 编码相同，按权重降序排列
		weightA := parseWeight(a.Weight)
		weightB := parseWeight(b.Weight)

		if weightA != weightB {
			return weightA > weightB
		}

		// 编码和权重都相同，按词语Unicode编码升序排列（保持稳定排序）
		return a.Word < b.Word
	})
}
