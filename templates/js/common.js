// 通用功能函数
// WebSocket连接
let socket = null;

function initWebSocket() {
    try {
        socket = new WebSocket('ws://' + window.location.host + '/ws');
        
        socket.onopen = function(event) {
            console.log('Connected to WebSocket');
        };
        
        socket.onmessage = function(event) {
            const logData = JSON.parse(event.data);
            const consoleDiv = document.getElementById('console');
            
            if (consoleDiv) {
                const timestamp = new Date().toLocaleTimeString();
                const lineDiv = document.createElement('div');
                lineDiv.className = 'console-line ' + logData.type;
                lineDiv.innerHTML = '<span class="text-gray-400">[' + timestamp + ']</span> ' +
                                   '<span class="text-blue-300">[' + logData.toolName + ']</span> ' +
                                   '<span class="' + (logData.type === 'error' ? 'text-red-400' : 
                                      logData.type === 'success' ? 'text-green-400 font-bold' : 
                                      'text-gray-300') + '">' + logData.message + '</span>';
                
                consoleDiv.appendChild(lineDiv);
                consoleDiv.scrollTop = consoleDiv.scrollHeight;
            }
        };
    } catch (e) {
        console.error('Failed to connect to WebSocket:', e);
    }
}

// 通用Tab切换功能
function switchTab(tabName) {
    // 隐藏所有标签内容
    const tabs = document.getElementsByClassName('tab-content');
    for (let i = 0; i < tabs.length; i++) {
        tabs[i].classList.add('hidden');
        tabs[i].classList.remove('active');
    }
    
    // 移除所有标签的激活状态
    const navTabs = document.getElementsByClassName('nav-tab');
    for (let i = 0; i < navTabs.length; i++) {
        navTabs[i].classList.remove('bg-blue-600', 'text-white');
        navTabs[i].classList.add('text-gray-200');
    }
    
    // 显示选中的标签内容
    const selectedTab = document.getElementById(tabName);
    if (selectedTab) {
        selectedTab.classList.remove('hidden');
        selectedTab.classList.add('active');
        
        // 更新选中的导航标签样式
        if (event && event.currentTarget) {
            event.currentTarget.classList.remove('text-gray-200');
            event.currentTarget.classList.add('bg-blue-600', 'text-white');
        }
    }
    
    // 如果切换到工具标签，则加载工具列表
    if (tabName === 'tools') {
        loadToolsList();
    }
    // 如果切换到文件管理标签，则加载文件列表
    else if (tabName === 'filemanager') {
        loadFileManager();
    }
    // 如果切换到章节管理标签，则加载章节列表
    else if (tabName === 'chapter-management') {
        loadChaptersList();
    }
}

// 通用API请求函数
async function apiRequest(url, options = {}) {
    try {
        const response = await fetch(url, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });
        return await response.json();
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    initWebSocket();
    
    // 设置视频背景
    setupVideoBackground();
    
    // 设置拖拽上传功能
    if (typeof initDragAndDrop !== 'undefined') {
        initDragAndDrop();
    }
    
    // 加载保存的设置
    if (typeof loadSavedSettings !== 'undefined') {
        loadSavedSettings();
    }
});

// 初始化 - 加载工具列表
window.onload = function() {
    // 在初始状态下加载工具列表
    if (document.querySelector('.nav-tab.active').textContent.includes('MCP 工具')) {
        loadToolsList();
    }

    // 加载风格模板选择器
    loadStyleTemplates();
    
    // 加载保存的设置
    loadSavedSettings();
};

// 设置视频背景
function setupVideoBackground() {
    const video = document.getElementById("myVideo");
    if (!video) return;
    
    const videos = ["assets/m1.mp4", "assets/m2.mp4", "assets/m3.mp4", "assets/m4.mp4", "assets/m5.mp4","assets/m6.mp4"];
    let currentIndex = Math.floor(Math.random() * videos.length); // 随机选择起始视频

    // 随机播放下一个视频
    video.addEventListener("ended", function() {
        let previousIndex = currentIndex;
        // 随机选择下一个视频，确保不重复播放同一个视频
        do {
            currentIndex = Math.floor(Math.random() * videos.length);
        } while (videos.length > 1 && currentIndex === previousIndex);
        
        video.src = videos[currentIndex];
        video.load(); // 重新加载新视频源
        video.play();
    });

    // 初始化随机播放第一个视频
    video.src = videos[currentIndex];
    video.play();
}

// 模态框相关函数
function closeModal() {
    const overlay = document.getElementById('modalOverlay');
    if (overlay) {
        overlay.remove();
    }
}

// 转义HTML以防止XSS
function escapeHtml(text) {
    var div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 一键出片功能
function oneClickFilm() {
    // 从localStorage获取保存的风格设置
    const savedTemplateId = localStorage.getItem('selectedStyleTemplateId') || '';
    const savedThreadCount = localStorage.getItem('threadCount') || '';
    
    fetch('/api/one-click-film', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            prompt_template_id: savedTemplateId || null,
            thread_count: savedThreadCount || null
        })
    })
    .then(response => response.json())
    .then(data => {
        console.log('一键出片工作流已启动:', data);
    })
    .catch(error => {
        console.error('一键出片执行错误:', error);
    });
}

// 停止执行功能
function stopExecution() {
    fetch('/api/stop', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({})
    })
    .then(response => response.json())
    .then(data => {
        console.log('Execution stopped:', data);
    })
    .catch(error => {
        console.error('Error stopping execution:', error);
    });
}

// 拖拽上传处理
function handleDragOver(e) {
    e.preventDefault();
    e.stopPropagation();
    document.getElementById('uploadArea').classList.add('border-blue-400', 'bg-blue-400', 'bg-opacity-10');
}

// 处理拖拽释放
function handleDrop(e) {
    e.preventDefault();
    e.stopPropagation();
    document.getElementById('uploadArea').classList.remove('border-blue-400', 'bg-blue-400', 'bg-opacity-10');
    
    const files = e.dataTransfer.files;
    if (files.length > 0) {
        // 简单显示上传状态
        document.getElementById('uploadStatus').innerText = '已放置文件: ' + files.length + ' 个文件';
    }
}

// 处理文件夹处理
function processFolder() {
    fetch('/api/process-folder', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({})
    })
    .then(response => response.json())
    .then(data => {
        console.log('Folder processing initiated:', data);
    })
    .catch(error => {
        console.error('Error processing folder:', error);
    });
}