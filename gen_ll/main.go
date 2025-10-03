package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"

	"gen_ll/tools"
	"gen_ll/types"
	"gen_ll/utils"
)

type Args struct {
	Quiet      bool   `flag:"q" usage:"安静模式，不输出进度信息" default:"false"`
	Div        string `flag:"d" usage:"拆分表文件"  default:"../deploy/hao/ll_div.txt"`
	Map        string `flag:"m" usage:"映射表文件"  default:"../deploy/hao/ll_map.txt"`
	Freq       string `flag:"f" usage:"频率表文件"  default:"../deploy/hao/freq.txt"`
	Words      string `flag:"w" usage:"多字词文件"  default:"../deploy/hao/ll_words.txt"`
	Full       string `flag:"u" usage:"输出全码表文件" default:"/tmp/code_full.txt"`
	Opencc     string `flag:"o" usage:"输出拆分表文件"  default:"/tmp/div.txt"`
	Simple     string `flag:"s" usage:"输出单字简码表文件" default:"/tmp/code_simp.txt"`
	WordsFull  string `flag:"W" usage:"输出多字词全码表文件" default:"/tmp/words_full.txt"`
	WordsSimple string `flag:"S" usage:"输出多字词简码表文件" default:"/tmp/words_simp.txt"`
	DazhuChai  string `flag:"Z" usage:"输出大竹拆文件" default:"/tmp/dazhu_chai.txt"`
	LenCodeLimit string `flag:"l" usage:"单字简码长度限制，格式：1:4,2:4,3:0,4:0" default:"1:4,2:4,3:0,4:0"`
	WordsLenCodeLimit string `flag:"L" usage:"多字词简码长度限制，格式：1:4,2:4,3:4,4:0" default:"1:4,2:4,3:4,4:0"`
	CPUProfile string `flag:"p" usage:"CPU性能分析文件" default:"/tmp/gen_ll.prof"`
	Debug      bool   `flag:"D" usage:"调试模式" default:"false"`
}

var args Args

func main() {
	err := utils.ParseFlags(&args)
	if err != nil {
		log.Fatalf("解析参数失败: %v", err)
		return
	}

	// CPU性能分析
	if args.CPUProfile != "" {
		f, err := os.Create(args.CPUProfile)
		if err != nil {
			log.Fatalf("无法创建CPU性能分析文件: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("无法开始CPU性能分析: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	// 创建输出目录（如果不存在）
	ensureOutputDir(args.Full)
	ensureOutputDir(args.Opencc)
	ensureOutputDir(args.Simple)
	ensureOutputDir(args.WordsFull)
	ensureOutputDir(args.WordsSimple)
	ensureOutputDir(args.DazhuChai)

	// 解析简码长度限制
	lenCodeLimit, err := tools.ParseLenCodeLimit(args.LenCodeLimit)
	if err != nil {
		log.Fatalf("解析单字简码长度限制失败: %v", err)
	}

	// 解析多字词简码长度限制
	wordsLenCodeLimit, err := tools.ParseLenCodeLimit(args.WordsLenCodeLimit)
	if err != nil {
		log.Fatalf("解析多字词简码长度限制失败: %v", err)
	}

	// 记录开始时间
	startTime := utils.Now()

	if !args.Quiet {
		fmt.Println("开始加载表格数据...")
	}

	divTable, err := tools.ReadDivisionTable(args.Div)
	if err != nil {
		log.Fatalf("读取拆分表失败: %v", err)
	}
	if !args.Quiet {
		fmt.Printf("拆分表加载完成，共 %d 项\n", len(divTable))
	}

	compMap, err := tools.ReadCompMap(args.Map)
	if err != nil {
		log.Fatalf("读取映射表失败: %v", err)
	}
	if !args.Quiet {
		fmt.Printf("映射表加载完成，共 %d 项\n", len(compMap))
	}

	// 验证拆分部件是否在映射表中定义
	if !args.Quiet {
		fmt.Println("开始验证拆分部件...")
	}
	if err := tools.ValidateDivisionComponents(divTable, compMap); err != nil {
		log.Fatalf("验证失败: %v", err)
	}
	if !args.Quiet {
		fmt.Println("拆分部件验证通过")
	}

	freqSet, err := tools.ReadCharFreq(args.Freq)
	if err != nil {
		log.Fatalf("读取频率表失败: %v", err)
	}
	if !args.Quiet {
		fmt.Printf("频率表加载完成，共 %d 项\n", len(freqSet))
	}

	if !args.Quiet {
		fmt.Println("开始构建编码数据...")
	}

	buildStartTime := utils.Now()
	fullCodeMetaList := tools.BuildFullCodeMetaList(divTable, compMap, freqSet)
	
	if !args.Quiet {
		fmt.Printf("构建完成，耗时: %v\n", utils.Since(buildStartTime))
		fmt.Printf("fullCodeMetaList: %d\n", len(fullCodeMetaList))
		fmt.Println("开始写入文件...")
	}

	// 读取多字词文件并生成多字词全码和简码
	var wordCodes []*types.WordCode
	var wordSimpleCodes []*types.WordSimpleCode
	if !args.Quiet {
		fmt.Println("开始读取多字词文件...")
	}
	wordEntries, err := tools.ReadWordsFile(args.Words)
	if err != nil {
		log.Printf("读取多字词文件失败: %v", err)
	} else {
		if !args.Quiet {
			fmt.Printf("多字词文件加载完成，共 %d 项\n", len(wordEntries))
			fmt.Println("开始生成多字词全码...")
		}
		
		// 创建字符编码映射
		charCodeMap := tools.CreateCharCodeMap(fullCodeMetaList)
		
		// 生成多字词全码
		wordCodes = tools.BuildWordsFullCode(wordEntries, charCodeMap)
		
		if !args.Quiet {
			fmt.Printf("多字词全码生成完成，共 %d 项\n", len(wordCodes))
			fmt.Println("开始生成多字词简码...")
		}
		
		// 生成多字词简码
		wordSimpleCodes = tools.BuildWordsSimpleCode(wordCodes, wordsLenCodeLimit)
		
		if !args.Quiet {
			fmt.Printf("多字词简码生成完成，共 %d 项\n", len(wordSimpleCodes))
		}
	}

	// 生成简码表
	if !args.Quiet {
		fmt.Println("开始生成简码表...")
	}
	noSimplifyChars := []string{"的", "了"} // 不出简的字符列表
	simpleCodeList := tools.BuildSimpleCodeList(fullCodeMetaList, lenCodeLimit, noSimplifyChars)
	
	if !args.Quiet {
		fmt.Printf("简码表生成完成，共 %d 项\n", len(simpleCodeList))
		fmt.Println("开始写入文件...")
	}

	// 使用并行处理加速文件写入
	var wg sync.WaitGroup
	fileCount := 4 // 基础文件：FULLCHAR, SIMPLECODE, DIVISION, DAZHUCHAI
	if wordCodes != nil {
		fileCount++
	}
	if wordSimpleCodes != nil {
		fileCount++
	}
	wg.Add(fileCount)
	errChan := make(chan error, fileCount)

	// FULLCHAR - 全码表，格式为"汉字\t编码\t词频"
	go func() {
		defer wg.Done()
		buffer := bytes.Buffer{}
		// 全码表已经在BuildFullCodeMetaList中排序过
		for _, charMeta := range fullCodeMetaList {
			buffer.WriteString(fmt.Sprintf("%s\t%s\t%d\n", charMeta.Char, charMeta.Code, charMeta.Freq))
		}
		err := os.WriteFile(args.Full, buffer.Bytes(), 0o644)
		if err != nil {
			errChan <- fmt.Errorf("写入FULLCHAR文件错误: %w", err)
		} else if !args.Quiet {
			fmt.Printf("FULLCHAR文件写入完成: %s\n", args.Full)
		}
	}()

	// SIMPLECODE
	go func() {
		defer wg.Done()
		buffer := bytes.Buffer{}
		// 对简码表进行排序：编码升序，重码按词频降序
		sortedSimpleList := make([]*types.CharMeta, len(simpleCodeList))
		copy(sortedSimpleList, simpleCodeList)
		sort.Slice(sortedSimpleList, func(i, j int) bool {
			a, b := sortedSimpleList[i], sortedSimpleList[j]
			
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
		for _, charMeta := range sortedSimpleList {
			buffer.WriteString(fmt.Sprintf("%s\t%s\t%d\n", charMeta.Char, charMeta.Code, charMeta.Freq))
		}
		err := os.WriteFile(args.Simple, buffer.Bytes(), 0o644)
		if err != nil {
			errChan <- fmt.Errorf("写入SIMPLECODE文件错误: %w", err)
		} else if !args.Quiet {
			fmt.Printf("SIMPLECODE文件写入完成: %s\n", args.Simple)
		}
	}()

	// DIVISION
	go func() {
		defer wg.Done()
		buffer := bytes.Buffer{}
		// 创建一个副本用于排序，避免并发访问问题
		sortedList := make([]*types.CharMeta, len(fullCodeMetaList))
		copy(sortedList, fullCodeMetaList)
		sort.Slice(sortedList, func(i, j int) bool {
			return sortedList[i].Char < sortedList[j].Char
		})
		for _, charMeta := range sortedList {
			if charMeta.Division == nil {
				continue
			}
			div := strings.Join(charMeta.Division.Divs, "")
			buffer.WriteString(fmt.Sprintf("%s\t(%s·%s·%s·%s·%s)\n",
	   			charMeta.Char,
	   			div,
	   			charMeta.Full,
	   			charMeta.Division.Pin,
	   			charMeta.Division.Set,
	   			charMeta.Division.Unicode,
			))
		}
		err := os.WriteFile(args.Opencc, buffer.Bytes(), 0o644)
		if err != nil {
			errChan <- fmt.Errorf("写入DIVISION文件错误: %w", err)
		} else if !args.Quiet {
			fmt.Printf("DIVISION文件写入完成: %s\n", args.Opencc)
		}
	}()

	// DAZHUCHAI - 大竹拆文件，格式为三行：
	// 第一行："部件\t字"（将 Division.Divs 连接成字符串）
	// 第二行："Unicode类别\t字"（使用 Division.Set）
	// 第三行："Unicode编码\t字"（使用 Division.Unicode）
	go func() {
		defer wg.Done()
		buffer := bytes.Buffer{}
		// 创建一个副本用于排序，按字符Unicode顺序排序
		sortedList := make([]*types.CharMeta, len(fullCodeMetaList))
		copy(sortedList, fullCodeMetaList)
		sort.Slice(sortedList, func(i, j int) bool {
			return sortedList[i].Char < sortedList[j].Char
		})
		for _, charMeta := range sortedList {
			if charMeta.Division == nil {
				continue
			}
			// 第一行：部件\t字
			components := strings.Join(charMeta.Division.Divs, "")
			buffer.WriteString(fmt.Sprintf("%s\t%s\n", components, charMeta.Char))
			// 第二行：Unicode类别\t字
			buffer.WriteString(fmt.Sprintf("%s\t%s\n", charMeta.Division.Set, charMeta.Char))
			// 第三行：Unicode编码\t字
			buffer.WriteString(fmt.Sprintf("%s\t%s\n", charMeta.Division.Unicode, charMeta.Char))
		}
		err := os.WriteFile(args.DazhuChai, buffer.Bytes(), 0o644)
		if err != nil {
			errChan <- fmt.Errorf("写入DAZHUCHAI文件错误: %w", err)
		} else if !args.Quiet {
			fmt.Printf("DAZHUCHAI文件写入完成: %s\n", args.DazhuChai)
		}
	}()

	// 写入多字词全码表
	if wordCodes != nil {
		go func() {
			defer wg.Done()
			buffer := bytes.Buffer{}
			
			// 对多字词编码进行排序
			// 先按编码升序排列，编码相同时按权重降序排列
			sortedWordCodes := make([]*types.WordCode, len(wordCodes))
			copy(sortedWordCodes, wordCodes)
			tools.SortWordCodes(sortedWordCodes)
			
			for _, wordCode := range sortedWordCodes {
				if wordCode.Weight != "" {
					buffer.WriteString(fmt.Sprintf("%s\t%s\t%s\n", wordCode.Word, wordCode.Code, wordCode.Weight))
				} else {
					buffer.WriteString(fmt.Sprintf("%s\t%s\n", wordCode.Word, wordCode.Code))
				}
			}
			err := os.WriteFile(args.WordsFull, buffer.Bytes(), 0o644)
			if err != nil {
				errChan <- fmt.Errorf("写入多字词全码表文件错误: %w", err)
			} else if !args.Quiet {
				fmt.Printf("多字词全码表文件写入完成: %s\n", args.WordsFull)
			}
		}()
	}

	// 写入多字词简码表
	if wordSimpleCodes != nil {
		go func() {
			defer wg.Done()
			buffer := bytes.Buffer{}
			
			// 对多字词简码进行排序
			// 先按编码升序排列，编码相同时按权重降序排列
			sortedWordSimpleCodes := make([]*types.WordSimpleCode, len(wordSimpleCodes))
			copy(sortedWordSimpleCodes, wordSimpleCodes)
			tools.SortWordSimpleCodes(sortedWordSimpleCodes)
			
			for _, wordSimpleCode := range sortedWordSimpleCodes {
				if wordSimpleCode.Weight != "" {
					buffer.WriteString(fmt.Sprintf("%s\t%s\t%s\n", wordSimpleCode.Word, wordSimpleCode.Code, wordSimpleCode.Weight))
				} else {
					buffer.WriteString(fmt.Sprintf("%s\t%s\n", wordSimpleCode.Word, wordSimpleCode.Code))
				}
			}
			err := os.WriteFile(args.WordsSimple, buffer.Bytes(), 0o644)
			if err != nil {
				errChan <- fmt.Errorf("写入多字词简码表文件错误: %w", err)
			} else if !args.Quiet {
				fmt.Printf("多字词简码表文件写入完成: %s\n", args.WordsSimple)
			}
		}()
	}

	// 等待所有写入操作完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		log.Fatalln(err)
	}

	// 输出处理时间
	if !args.Quiet {
		fmt.Printf("处理完成，总耗时: %v\n", utils.Since(startTime))
	}
}

// 确保输出目录存在
func ensureOutputDir(path string) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("无法创建目录 %s: %v", dir, err)
		}
	}
}
