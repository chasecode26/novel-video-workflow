package file

import (
	"bufio"
	"fmt"
	"log"
	"novel-video-workflow/pkg/broadcast"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileManager struct {
	BroadcastService *broadcast.BroadcastService
}

func NewFileManager() *FileManager {
	return &FileManager{
		BroadcastService: broadcast.NewBroadcastService(),
	}
}

type ChapterStructure struct {
	ChapterDir  string
	TextFile    string
	AudioDir    string
	SubtitleDir string
	ImageDir    string
	SceneDir    string
}

// Comment 表示一个注释
type Comment struct {
	ID        string    `json:"id"`         // 注释唯一标识
	Content   string    `json:"content"`    // 注释内容
	Line      int       `json:"line"`       // 注释所在行号
	StartPos  int       `json:"start_pos"`  // 在行内的起始位置
	EndPos    int       `json:"end_pos"`    // 在行内的结束位置
	Type      string    `json:"type"`       // 注释类型 (info, warning, error, highlight等)
	CreatedAt time.Time `json:"created_at"` // 创建时间
	Author    string    `json:"author"`     // 注释作者
}

// CommentsCollection 存储文本的注释集合
type CommentsCollection struct {
	Filepath string    `json:"filepath"` // 关联的文件路径
	Comments []Comment `json:"comments"` // 注释列表
}

// ChapterContentMap 章节内容映射，键为章节号，值为章节内容
type ChapterContentMap map[int]string

var ChapterMap ChapterContentMap
var chapterMapMutex sync.Mutex // 保护ChapterMap的互斥锁

// 这里需要传递一个.txt的绝对路径
func (fm *FileManager) CreateInputChapterStructure(absDir string) (*ChapterStructure, error) {
	if c_map, err := fm.ExtractChapterTxt(absDir); err != nil {
		return nil, err
	} else {
		// 使用互斥锁保护ChapterMap的写入
		chapterMapMutex.Lock()
		ChapterMap = c_map
		chapterMapMutex.Unlock()

		// 循环c_map并创建文件夹，创建新的txt文本放到文件夹下
		for chapterNum, content := range c_map {
			fm.CreateChapterStructure(chapterNum, content, absDir)
		}
	}
	//构建input文件夹
	return nil, nil
}

// CreateChapterStructure 创建章节目录结构，格式为 chapter_XX/chapter_XX.txt
func (fm *FileManager) CreateChapterStructure(chapterNum int, content string, absDir string) error {
	// 获取基础目录路径
	basePath := filepath.Dir(absDir)

	// 格式化章节号，确保两位数格式（如 01, 02, ...）
	chapterFolderName := fmt.Sprintf("chapter_%02d", chapterNum)
	chapterFileName := fmt.Sprintf("chapter_%02d.txt", chapterNum)

	// 创建章节目录
	chapterDir := filepath.Join(basePath, chapterFolderName)
	err := os.MkdirAll(chapterDir, 0755)
	if err != nil {
		return fmt.Errorf("创建章节目录失败: %v", err)
	}

	// 创建章节文件
	filePath := filepath.Join(chapterDir, chapterFileName)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建章节文件失败: %v", err)
	}
	defer file.Close()

	// 写入内容
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("写入章节内容失败: %v", err)
	}

	return nil
}

// ExtractChapterTxt 提取章节编号和对应的内容，返回章节编号到内容的映射
func (fm *FileManager) ExtractChapterTxt(fileDir string) (ChapterContentMap, error) {
	fileHandle, err := os.OpenFile(fileDir, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	defer fileHandle.Close()

	chapterMap := make(ChapterContentMap)
	var currentContent strings.Builder
	currentChapterFound := false
	var currentChapterNum int

	scanner := bufio.NewScanner(fileHandle)
	// 兼容“第一章 标题”“第12章 标题”“第十二节 标题”等常见单文件小说章节头。
	re := regexp.MustCompile(`^\s*第\s*([0-9零一二三四五六七八九十百千万两〇]+)\s*([章节回节卷篇幕集])(?:\s+|[:：\-—_])?(.*)$`)

	for scanner.Scan() {
		text := scanner.Text()

		// 检查当前行是否为章节标记
		if matches := re.FindStringSubmatch(text); len(matches) > 0 {
			// 如果已经找到了上一个章节的内容，保存它
			if currentChapterFound {
				chapterMap[currentChapterNum] = strings.TrimSpace(currentContent.String())
				currentContent.Reset()
			}

			// 提取章节数字
			numStr := strings.TrimSpace(matches[1])

			// 转换为阿拉伯数字
			if atoi, err := strconv.Atoi(numStr); err != nil {
				currentChapterNum = fm.convertChineseNumberToArabic(numStr)
			} else {
				currentChapterNum = atoi
			}
			if currentChapterNum <= 0 {
				currentChapterNum = len(chapterMap) + 1
			}

			currentChapterFound = true

			// 将章节标题也加入内容中
			currentContent.WriteString(text)
			currentContent.WriteString("\n")
		} else {
			// 如果当前行不是章节标记，将其添加到当前内容中
			if currentChapterFound {
				currentContent.WriteString(text)
				currentContent.WriteString("\n")
			}
		}
	}

	// 处理最后一个章节的内容
	if currentChapterFound {
		chapterMap[currentChapterNum] = strings.TrimSpace(currentContent.String())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return chapterMap, nil
}

// ConvertChineseNumberToArabic 将中文数字转换为阿拉伯数字
func (fm *FileManager) convertChineseNumberToArabic(chineseNum string) int {
	chineseNum = strings.TrimSpace(chineseNum)
	if chineseNum == "" {
		return 0
	}

	if value, err := strconv.Atoi(chineseNum); err == nil {
		return value
	}

	digits := map[rune]int{
		'零': 0, '〇': 0,
		'一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9,
	}
	units := map[rune]int{
		'十': 10,
		'百': 100,
		'千': 1000,
		'万': 10000,
	}

	total := 0
	section := 0
	number := 0
	for _, char := range chineseNum {
		if value, ok := digits[char]; ok {
			number = value
			continue
		}
		unit, ok := units[char]
		if !ok {
			return 0
		}
		if unit == 10000 {
			if number == 0 && section == 0 {
				section = 1
			} else {
				section += number
			}
			total += section * unit
			section = 0
			number = 0
			continue
		}
		if number == 0 {
			number = 1
		}
		section += number * unit
		number = 0
	}

	return total + section + number
}

// output则参考input的结构生成目录结构，分出章节，每个章节内参考如下即可
/*
```
output/
└── 小说名称/
    └── chapter_01/
        ├── chapter_01.wav      # 音频文件
        ├── chapter_01.srt      # 字幕文件
        └── images/             # 图像目录
            ├── scene_01.png
            ├── scene_02.png
            └── ...
    └── chapter_02/
        ├── chapter_02.wav      # 音频文件
        ├── chapter_02.srt      # 字幕文件
        └── images/             # 图像目录
            ├── scene_01.png
            ├── scene_02.png
            └── ...
```
*/
func (fm *FileManager) CreateOutputChapterStructure(inpDir string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	//inpDir下的文件夹名字
	fold_name := ""
	items, err := os.ReadDir(inpDir)
	for _, item := range items {
		if item.IsDir() {
			fold_name = item.Name()
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	// 1、创建这个文件夹
	os.Mkdir(filepath.Join(dir, "output", fold_name), os.ModePerm)

	// 创建子文件夹
	// 使用互斥锁保护ChapterMap的读取
	chapterMapMutex.Lock()
	defer chapterMapMutex.Unlock()

	for key, _ := range ChapterMap {
		f_name := fmt.Sprintf("chapter_%02d", key)
		//创建文件夹
		os.Mkdir(filepath.Join(dir, "output", fold_name, f_name), os.ModePerm)
	}
	fmt.Println(dir)
}
