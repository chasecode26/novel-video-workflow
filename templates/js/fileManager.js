// 文件管理相关功能函数
let currentDirectory = 'input'; // 默认目录

// 切换目录
function changeDirectory(dir) {
    // 去掉 ./ 前缀，只保留目录名
    currentDirectory = dir.startsWith('./') ? dir.substring(2) : dir;
    loadFileList(currentDirectory);
}


// 加载文件列表
function loadFileList(dir) {
    // 构建正确的路径格式，确保后端能正确处理
    var pathToUse = dir;
    // 如果路径不以 ./ 开头，添加前缀
    if (!dir.startsWith('./')) {
        pathToUse = './' + dir;
    }

    fetch('/api/files/list?dir=' + encodeURIComponent(pathToUse))
        .then(response => response.json())
        .then(data => {
            const fileListDiv = document.getElementById('fileList');
            if (!fileListDiv) return;

            fileListDiv.innerHTML = '';

            if (data.files.length === 0) {
                fileListDiv.innerHTML = '<div class="text-center py-16 text-2xl text-gray-300"><i class="fas fa-inbox text-6xl mb-6 opacity-50"></i><p>目录为空</p></div>';
                return;
            }

            // 创建文件表格
            const table = document.createElement('table');
            table.className = 'min-w-full divide-y divide-gray-600';
            table.innerHTML = `
                <thead class="bg-black bg-opacity-20">
                    <tr>
                        <th class="px-6 py-5 text-left text-lg font-bold text-gray-300 uppercase tracking-wider rounded-tl-lg">名称</th>
                        <th class="px-6 py-5 text-left text-lg font-bold text-gray-300 uppercase tracking-wider">类型</th>
                        <th class="px-6 py-5 text-left text-lg font-bold text-gray-300 uppercase tracking-wider">大小</th>
                        <th class="px-6 py-5 text-left text-lg font-bold text-gray-300 uppercase tracking-wider">修改时间</th>
                        <th class="px-6 py-5 text-left text-lg font-bold text-gray-300 uppercase tracking-wider rounded-tr-lg">操作</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-gray-600"></tbody>
            `;

            const tbody = table.querySelector('tbody');

            data.files.forEach(function (file) {
                const row = document.createElement('tr');
                row.className = 'hover:bg-white hover:bg-opacity-10 transition-colors duration-150';

                var fileTypeDisplay = file.isDir ? '📁' : getFileIconByType(file.type);
                var fileSizeDisplay = file.isDir ? '-' : formatFileSize(file.size);
                var previewButton = '';
                // 扩展预览功能到图片和音频文件
                if (!file.isDir && ['text', 'json', 'yaml', 'yml', 'xml', 'csv', 'log', 'md', 'image', 'audio'].includes(file.type)) {
                    previewButton = '<button onclick="previewFile(\'' + file.name + '\')" class="text-blue-300 hover:text-blue-100 mr-5 text-lg transition-colors duration-150"><i class="fas fa-eye mr-2"></i>预览</button>';
                }
                var modTime = new Date(file.modTime).toLocaleString();

                row.innerHTML = "<td class=\"px-6 py-6 whitespace-nowrap\">\n                                <div class=\"flex items-center\">\n                                    <div class=\"mr-5 text-2xl opacity-80\">" + fileTypeDisplay + "</div>\n                                    <div class=\"text-xl font-medium text-gray-200\">\n                                        <span onclick=\"clickFileOrDir('" + file.name + "', " + file.isDir + ")\" class=\"cursor-pointer hover:text-blue-300 transition-colors duration-150\">" + file.name + "</span>\n                                    </div>\n                                </div>\n                            </td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl text-gray-300\">" + file.type + "</td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl text-gray-300\">" + fileSizeDisplay + "</td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl text-gray-300\">" + modTime + "</td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl font-medium\">\n                                " + previewButton + "\n                                " + (file.isDir && file.name.match(/^chapter_\d+$/) && (currentDirectory.includes('/output/') || currentDirectory.includes('output')) ? '<button onclick="sendToCapcut(\'' + file.name + '\')" class="text-green-400 hover:text-green-200 mr-5 text-lg transition-colors duration-150"><i class="fas fa-share mr-2"></i>一键到剪映</button>' : '') + "\n                                <button onclick=\"deleteFile('" + file.name + "', " + file.isDir + ")\" class=\"text-red-400 hover:text-red-200 transition-colors duration-150\"><i class=\"fas fa-trash mr-2\"></i>删除</button>\n                            </td>\n                        ";
                tbody.appendChild(row);
            });

            fileListDiv.appendChild(table);
        })
        .catch(function (error) {
            console.error('Error loading file list:', error);
            const fileListDiv = document.getElementById('fileList');
            if (fileListDiv) {
                fileListDiv.innerHTML = '<div class="text-red-400 p-8 text-center text-xl"><i class="fas fa-exclamation-triangle mr-3"></i>加载文件列表失败: ' + error.message + '</div>';
            }
        });
}

// 根据文件类型获取图标
function getFileIconByType(fileType) {
    switch (fileType) {
        case 'image':
            return '🖼️';
        case 'video':
            return '🎬';
        case 'audio':
            return '🎵';
        case 'pdf':
            return '📄';
        case 'archive':
            return '📦';
        case 'text':
        case 'json':
        case 'yaml':
        case 'yml':
        case 'xml':
        case 'csv':
        case 'log':
        case 'md':
            return '📝';
        default:
            return '📄';
    }
}

// 格式化文件大小
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 点击文件或目录
function clickFileOrDir(name, isDir) {
    if (isDir) {
        // 进入子目录
        var newPath = currentDirectory;
        if (newPath.endsWith('/')) {
            newPath += name;
        } else {
            newPath += '/' + name;
        }
        changeDirectory('./' + newPath);
    } else {
        // 对于支持预览的文件类型，打开预览
        previewFile(name);
    }
}

// 预览文件
function previewFile(filename) {
    const fullPath = './' + currentDirectory + '/' + filename;

    // 根据文件扩展名判断文件类型
    const ext = filename.split('.').pop().toLowerCase();
    const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'];
    const audioExtensions = ['mp3', 'wav', 'ogg', 'm4a', 'flac'];
    const textExtensions = ['txt', 'json', 'yaml', 'yml', 'xml', 'csv', 'log', 'md'];
    const subtitleExtensions = ['srt', 'ass', 'vtt']; // 字幕文件

    if (imageExtensions.includes(ext)) {
        // 图片预览 - 使用正确的静态文件路径
        const staticPath = convertToStaticPath(fullPath);
        showImagePreview(staticPath, filename);
    } else if (audioExtensions.includes(ext)) {
        // 音频预览 - 使用正确的静态文件路径
        const staticPath = convertToStaticPath(fullPath);
        showAudioPreview(staticPath, filename);
    } else if (subtitleExtensions.includes(ext)) {
        // 字幕文件预览（通过API获取内容）
        fetch('/api/files/content?path=' + encodeURIComponent(fullPath))
            .then(response => response.text())
            .then(content => {
                showTextPreview(content, filename);
            })
            .catch(function (error) {
                console.error('Error previewing subtitle file:', error);
                alert('无法预览字幕文件: ' + error.message);
            });
    } else if (textExtensions.includes(ext)) {
        // 文本文件预览（通过API获取内容）
        fetch('/api/files/content?path=' + encodeURIComponent(fullPath))
            .then(response => response.text())
            .then(content => {
                showTextPreview(content, filename);
            })
            .catch(function (error) {
                console.error('Error previewing text file:', error);
                alert('无法预览文件: ' + error.message);
            });
    } else {
        // 对于其他文件类型，尝试通过API获取内容
        fetch('/api/files/content?path=' + encodeURIComponent(fullPath))
            .then(response => {
                if (response.ok) {
                    // 如果是文本内容则显示为文本，否则提示无法预览
                    return response.text().then(content => {
                        showTextPreview(content, filename);
                    });
                } else {
                    throw new Error('HTTP error! status: ' + response.status);
                }
            })
            .catch(function (error) {
                console.error('Error previewing file:', error);
                alert('无法预览文件: ' + error.message);
            });
    }
}

// 将相对路径转换为静态文件服务路径
function convertToStaticPath(relativePath) {
    // 将 ./input/path 或 ./output/path 转换为 /files/input/path 或 /files/output/path
    if (relativePath.startsWith('./input/')) {
        return '/files/input/' + relativePath.substring(9); // 9 = length of './input/'
    } else if (relativePath.startsWith('./output/')) {
        return '/files/output/' + relativePath.substring(9); // 9 = length of './output/'
    }
    return relativePath; // 如果不是标准路径，返回原始路径
}

// 显示文本预览
function showTextPreview(content, filename) {
    // 创建模态框显示文件内容
    const overlay = document.createElement('div');
    overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
    overlay.id = 'modalOverlay';

    const modal = document.createElement('div');
    modal.className = 'glass-effect rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col border border-white border-opacity-30';

    const header = document.createElement('div');
    header.className = 'flex justify-between items-center p-6 border-b border-white border-opacity-30 rounded-t-2xl';
    header.innerHTML = '<h3 class="text-xl font-bold text-white">预览: ' + filename + '</h3>' +
        '<button onclick="closeModal()" class="text-gray-300 hover:text-white text-3xl leading-none">' +
        '<i class="fas fa-times"></i>' +
        '</button>';

    const contentDiv = document.createElement('div');
    contentDiv.className = 'p-6 flex-grow overflow-auto';
    contentDiv.innerHTML = '<pre class="whitespace-pre-wrap break-words font-mono text-lg text-gray-200 bg-black bg-opacity-30 p-6 rounded-xl">' + escapeHtml(content) + '</pre>';

    modal.appendChild(header);
    modal.appendChild(contentDiv);
    overlay.appendChild(modal);
    document.body.appendChild(overlay);
}

// 显示图片预览
function showImagePreview(filePath, filename) {
    // 创建模态框显示图片
    const overlay = document.createElement('div');
    overlay.className = 'fixed inset-0 bg-black bg-opacity-90 flex items-center justify-center z-50 p-4';
    overlay.id = 'modalOverlay';

    const modal = document.createElement('div');
    modal.className = 'w-full max-w-6xl max-h-[90vh] flex flex-col';

    const header = document.createElement('div');
    header.className = 'flex justify-between items-center p-4 bg-black bg-opacity-30 rounded-t-2xl';
    header.innerHTML = '<h3 class="text-xl font-bold text-white">预览: ' + filename + '</h3>' +
        '<button onclick="closeModal()" class="text-gray-300 hover:text-white text-3xl leading-none">' +
        '<i class="fas fa-times"></i>' +
        '</button>';

    const contentDiv = document.createElement('div');
    contentDiv.className = 'flex-grow flex items-center justify-center p-4';
    contentDiv.innerHTML = '<img src="' + filePath + '" alt="' + filename + '" class="max-h-[70vh] max-w-full object-contain rounded-xl">';

    modal.appendChild(header);
    modal.appendChild(contentDiv);
    overlay.appendChild(modal);
    document.body.appendChild(overlay);
}

// 显示音频预览
function showAudioPreview(filePath, filename) {
    // 创建模态框显示音频播放器
    const overlay = document.createElement('div');
    overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
    overlay.id = 'modalOverlay';

    const modal = document.createElement('div');
    modal.className = 'glass-effect rounded-2xl w-full max-w-2xl flex flex-col border border-white border-opacity-30';

    const header = document.createElement('div');
    header.className = 'flex justify-between items-center p-6 border-b border-white border-opacity-30 rounded-t-2xl';
    header.innerHTML = '<h3 class="text-xl font-bold text-white">播放: ' + filename + '</h3>' +
        '<button onclick="closeModal()" class="text-gray-300 hover:text-white text-3xl leading-none">' +
        '<i class="fas fa-times"></i>' +
        '</button>';

    const contentDiv = document.createElement('div');
    contentDiv.className = 'p-8 flex flex-col items-center';
    contentDiv.innerHTML = '<div class="w-full max-w-md">' +
        '<div class="text-center mb-6">' +
        '<i class="fas fa-music text-6xl text-blue-400 mb-4"></i>' +
        '<p class="text-xl text-white">' + filename + '</p>' +
        '</div>' +
        '<audio controls class="w-full h-12 rounded-lg">' +
        '<source src="' + filePath + '" type="audio/' + filename.split('.').pop() + '">' +
        '您的浏览器不支持音频播放。' +
        '</audio>' +
        '</div>';

    modal.appendChild(header);
    modal.appendChild(contentDiv);
    overlay.appendChild(modal);
    document.body.appendChild(overlay);
}

// 删除文件或目录
function deleteFile(filename, isDir) {
    var confirmMessage = '确定要删除' + (isDir ? '目录' : '文件') + ' "' + filename + '" 吗？此操作不可撤销。';
    if (confirm(confirmMessage)) {
        const fullPath = './' + currentDirectory + '/' + filename;
        fetch('/api/files/delete?path=' + encodeURIComponent(fullPath), {
            method: 'DELETE'
        })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    alert((isDir ? '目录' : '文件') + ' 已成功删除');
                    loadFileList(currentDirectory); // 重新加载文件列表
                } else {
                    alert('删除失败: ' + data.message);
                }
            })
            .catch(function (error) {
                console.error('Error deleting file:', error);
                alert('删除失败: ' + error.message);
            });
    }
}

// 发送到剪映功能
function sendToCapcut(folderName) {
    if (!folderName.match(/^chapter_\d+$/)) {
        alert('请选择正确的章节目录（格式为chapter_XX）');
        return;
    }

    // 构建完整的路径
    const fullPath = './' + currentDirectory + '/' + folderName;

    if (confirm('确定要将 ' + folderName + ' 导入到剪映吗？')) {
        fetch('/api/capcut-project?chapter_path=' + encodeURIComponent(fullPath))
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    alert('已开始生成剪映项目，请查看控制台进度');
                } else {
                    alert('操作失败: ' + (data.error || '未知错误'));
                }
            })
            .catch(error => {
                console.error('Error:', error);
                alert('操作失败: ' + error.message);
            });
    }
}


// 添加拖拽上传功能
function initDragAndDrop() {
    const dropArea = document.getElementById('folderUploadArea');

    if (dropArea) {
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            dropArea.addEventListener(eventName, preventDefaults, false);
        });

        function preventDefaults(e) {
            e.preventDefault();
            e.stopPropagation();
        }

        ['dragenter', 'dragover'].forEach(eventName => {
            dropArea.addEventListener(eventName, highlight, false);
        });

        ['dragleave', 'drop'].forEach(eventName => {
            dropArea.addEventListener(eventName, unhighlight, false);
        });

        function highlight() {
            dropArea.style.borderColor = '#3b82f6';
            dropArea.style.backgroundColor = 'rgba(59, 130, 246, 0.1)';
        }

        function unhighlight() {
            dropArea.style.borderColor = '#9ca3af';
            dropArea.style.backgroundColor = 'transparent';
        }

        dropArea.addEventListener('drop', handleDrop, false);

        function handleDrop(e) {
            const dt = e.dataTransfer;
            const files = dt.files;

            // 处理拖放的文件夹
            handleFolderUpload(files);
        }
    }
}
// 在页面完全加载后初始化拖拽功能
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initDragAndDrop);
} else {
    // 如果文档已经加载完成，则直接初始化
    initDragAndDrop();
}

// 当切换到文件管理标签时加载文件列表
function loadFileManager() {
    // 使用input作为默认目录
    loadFileList('input');
}

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
    document.getElementById(tabName).classList.remove('hidden');
    document.getElementById(tabName).classList.add('active');

    // 更新选中的导航标签样式
    event.currentTarget.classList.remove('text-gray-200');
    event.currentTarget.classList.add('bg-blue-600', 'text-white');

    // 如果切换到工具标签，则加载工具列表
    if (tabName === 'tools') {
        loadToolsList();
    }
    // 如果切换到文件管理标签，则加载文件列表
    else if (tabName === 'filemanager') {
        loadFileManager();
    }
}


function loadToolsList() {
    fetch('/api/tools')
        .then(response => response.json())
        .then(tools => {
            const toolsListDiv = document.getElementById('tools-list');
            toolsListDiv.innerHTML = '';

            // 不同工具的图标映射
            const toolIcons = {
                'generate_indextts2_audio': 'fas fa-microphone-alt',
                'generate_subtitles_from_indextts2': 'fas fa-closed-captioning',
                'file_split_novel_into_chapters': 'fas fa-file-alt',
                'generate_image_from_text': 'fas fa-paint-brush',
                'generate_image_from_image': 'fas fa-image',
                'generate_images_from_chapter': 'fas fa-images',
                'generate_images_from_chapter_with_ai_prompt': 'fas fa-robot',
                'generate_image_from_lyric_ai_prompt': 'fas fa-music', // 歌词工具图标
                'default': 'fas fa-cog'
            };

            tools.forEach(function (tool) {
                const toolCard = document.createElement('div');
                toolCard.className = 'glass-effect rounded-2xl p-6 border border-white border-opacity-20 card-hover future-glow';

                // 获取对应工具的图标
                const iconClass = toolIcons[tool.name] || toolIcons['default'];

                let cardContent = '';

                // 特殊处理歌词MV生成工具 - 确保名称完全匹配
                if (tool.name === 'generate_image_from_lyric_ai_prompt') {
                    console.log('匹配到歌词工具:', tool.name); // 调试信息
                    cardContent = `
                        <div class="tool-header text-center">
                            <div class="flex flex-col items-center">
                                <div class="w-20 h-20 bg-purple-500 bg-opacity-20 rounded-full flex items-center justify-center mb-4">
                                    <i class="fas fa-music text-purple-300 text-2xl"></i>
                                </div>
                                <h3 class="text-xl font-bold text-white mb-2 break-words max-w-full">${tool.name}</h3>
                                <p class="text-lg text-gray-300 mb-6">${tool.description}</p>
                                <button onclick="toggleLyricForm('${tool.name}')" class="w-full bg-gradient-to-r from-purple-500 to-purple-600 hover:from-purple-600 hover:to-purple-700 text-white py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">
                                    执行工具
                                </button>
                            </div>
                        </div>
                        <div id="form_${tool.name}" class="lyric-tool-form mt-6 p-6 glass-effect rounded-xl border border-white border-opacity-20 hidden">
                            <div class="form-group mb-6">
                                <label class="block text-lg font-bold text-gray-300 mb-3">歌词文本:</label>
                                <textarea id="lyricText_${tool.name}" placeholder="请输入歌词文本，每行一句歌词" rows="6" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500 text-lg"></textarea>
                                <p class="mt-2 text-sm text-gray-400">支持多行歌词输入，AI将为每一句歌词生成对应的MV画面</p>
                            </div>
                            <div class="form-group mb-6">
                                <label class="block text-lg font-bold text-gray-300 mb-3">输出目录:</label>
                                <input type="text" id="outputDir_${tool.name}" value="./output/lyric_mv_" placeholder="请输入输出目录" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500 text-lg">
                            </div>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                                <div class="form-group">
                                    <label class="block text-lg font-bold text-gray-300 mb-3">图像宽度:</label>
                                    <input type="number" id="imageWidth_${tool.name}" value="512" min="256" max="2048" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500 text-lg">
                                </div>
                                <div class="form-group">
                                    <label class="block text-lg font-bold text-gray-300 mb-3">图像高度:</label>
                                    <input type="number" id="imageHeight_${tool.name}" value="896" min="256" max="2048" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500 text-lg">
                                </div>
                            </div>
                            <div class="flex flex-wrap gap-4">
                                <button onclick="executeLyricTool('${tool.name}')" class="bg-gradient-to-r from-purple-500 to-purple-600 hover:from-purple-600 hover:to-purple-700 text-white px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">
                                    <i class="fas fa-play mr-2"></i>生成MV图像
                                </button>
                                <button onclick="hideLyricForm('${tool.name}')" class="bg-gradient-to-r from-gray-500 to-gray-600 hover:from-gray-600 hover:to-gray-700 text-white px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">
                                    取消
                                </button>
                            </div>
                        </div>
                    `;
                }
                // 为generate_indextts2_audio工具添加特殊处理
                else if (tool.name === 'generate_indextts2_audio') {
                    // 为音频生成工具添加表单
                    cardContent = `
                        <div class="tool-header text-center">
                            <div class="flex flex-col items-center">
                                <div class="w-20 h-20 bg-blue-500 bg-opacity-20 rounded-full flex items-center justify-center mb-4">
                                    <i class="${iconClass} text-blue-300 text-2xl"></i>
                                </div>
                                <h3 class="text-xl font-bold text-white mb-3">${tool.name}</h3>
                                <p class="text-lg text-gray-300 mb-6">${tool.description}</p>
                                <button onclick="toggleAudioForm('${tool.name}')" class="w-full bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">
                                    执行工具
                                </button>
                            </div>
                        </div>
                        <div id="form_${tool.name}" class="audio-tool-form mt-6 p-6 glass-effect rounded-xl border border-white border-opacity-20 hidden">
                            <div class="form-group mb-6">
                                <label class="block text-lg font-bold text-gray-300 mb-3">输入文本:</label>
                                <textarea id="textInput_${tool.name}" placeholder="请输入要转换为语音的文本" rows="3" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg"></textarea>
                            </div>
                            <div class="form-group mb-6">
                                <label class="block text-lg font-bold text-gray-300 mb-3">输出目录:</label>
                                <input type="text" id="outputDir_${tool.name}" value="./output/" placeholder="请输入输出目录" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">
                            </div>
                            <div class="flex flex-wrap gap-4">
                                <button onclick="executeAudioTool('${tool.name}')" class="bg-gradient-to-r from-green-500 to-green-600 hover:from-green-600 hover:to-green-700 text-white px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">
                                    生成音频
                                </button>
                                <button onclick="hideAudioForm('${tool.name}')" class="bg-gradient-to-r from-gray-500 to-gray-600 hover:from-gray-600 hover:to-gray-700 text-white px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">
                                    取消
                                </button>
                            </div>
                        </div>
                    `;
                } else if (tool.name === 'generate_images_from_chapter_with_ai_prompt') {
                    // 为图像生成工具添加表单
                    cardContent = `
                        <div class="tool-header text-center">
                            <div class="flex flex-col items-center">
                                <div class="w-20 h-20 bg-blue-500 bg-opacity-20 rounded-full flex items-center justify-center mb-4">
                                    <i class="${iconClass} text-blue-300 text-2xl"></i>
                                </div>
                                <h3 class="text-xl font-bold text-white mb-2 break-words max-w-full">${tool.name}</h3>
                                <p class="text-lg text-gray-300 mb-6">${tool.description}</p>
                                <button onclick="toggleImageForm('${tool.name}')" class="w-full bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">
                                    执行工具
                                </button>
                            </div>
                        </div>
                        <div id="form_${tool.name}" class="audio-tool-form mt-6 p-6 glass-effect rounded-xl border border-white border-opacity-20 hidden">
                            <div class="form-group mb-6">
                                <label class="block text-lg font-bold text-gray-300 mb-3">章节文本:</label>
                                <textarea id="chapterText_${tool.name}" placeholder="请输入要生成图像的章节文本" rows="4" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg"></textarea>
                            </div>
                            <div class="form-group mb-6">
                                <label class="block text-lg font-bold text-gray-300 mb-3">输出目录:</label>
                                <input type="text" id="outputDir_${tool.name}" value="./output/images_" placeholder="请输入输出目录" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">
                            </div>
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                                <div class="form-group">
                                    <label class="block text-lg font-bold text-gray-300 mb-3">图像宽度:</label>
                                    <input type="number" id="imageWidth_${tool.name}" value="512" min="256" max="2048" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">
                                </div>
                                <div class="form-group">
                                    <label class="block text-lg font-bold text-gray-300 mb-3">图像高度:</label>
                                    <input type="number" id="imageHeight_${tool.name}" value="896" min="256" max="2048" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">
                                </div>
                            </div>
                            <div class="flex flex-wrap gap-4">
                                <button onclick="executeImageTool('${tool.name}')" class="bg-gradient-to-r from-gray-200 to-gray-300 hover:from-gray-300 hover:to-gray-400 text-gray-800 px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">
                                    生成图像
                                </button>
                                <button onclick="hideImageForm('${tool.name}')" class="bg-gradient-to-r from-gray-200 to-gray-300 hover:from-gray-300 hover:to-gray-400 text-gray-800 px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">
                                    取消
                                </button>
                            </div>
                        </div>
                    `;
                } else {
                    // 其他工具使用普通按钮
                    cardContent = `
                        <div class="tool-header text-center">
                            <div class="flex flex-col items-center">
                                <div class="w-20 h-20 bg-blue-500 bg-opacity-20 rounded-full flex items-center justify-center mb-5">
                                    <i class="${iconClass} text-blue-300 text-3xl"></i>
                                </div>
                                <h3 class="text-xl font-bold text-white mb-3">${tool.name}</h3>
                                <p class="text-lg text-gray-300 mb-6">${tool.description}</p>
                                <button onclick="executeTool('${tool.name}')" class="w-full bg-gradient-to-r from-gray-200 to-gray-300 hover:from-gray-300 hover:to-gray-400 text-gray-800 py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">
                                    执行工具
                                </button>
                            </div>
                        </div>
                    `;
                }

                toolCard.innerHTML = cardContent;
                toolsListDiv.appendChild(toolCard);
            });
        })
        .catch(error => {
            console.error('Error loading tools:', error);
            // 添加错误提示
            const toolsListDiv = document.getElementById('tools-list');
            toolsListDiv.innerHTML = '<div class="col-span-full text-red-400 p-8 text-center text-xl"><i class="fas fa-exclamation-triangle mr-3"></i>加载工具列表失败: ' + error.message + '</div>';
        });
}



function executeLyricTool(toolName) {
    const lyricText = document.getElementById('lyricText_' + toolName).value;
    const outputDir = document.getElementById('outputDir_' + toolName).value;
    const imageWidth = parseInt(document.getElementById('imageWidth_' + toolName).value);
    const imageHeight = parseInt(document.getElementById('imageHeight_' + toolName).value);

    if (!lyricText || lyricText.trim() === '') {
        alert('请输入歌词文本');
        return;
    }

    if (!outputDir || outputDir.trim() === '') {
        alert('请输入输出目录');
        return;
    }

    // 生成输出目录路径
    const timestamp = new Date().getTime();
    const outputDirectory = outputDir + timestamp;

    const params = {
        toolName: toolName,
        lyric_text: lyricText,
        output_dir: outputDirectory,
        width: imageWidth,
        height: imageHeight
    };

    fetch('/api/execute', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(params)
    })
        .then(response => response.json())
        .then(data => {
            console.log('Lyric tool execution initiated:', data);
            // 隐藏表单
            hideLyricForm(toolName);

            // 显示成功提示
            showNotification('歌词MV图像生成已启动，请在控制台查看进度', 'success');
        })
        .catch(error => {
            console.error('Error executing lyric tool:', error);
            showNotification('执行歌词工具时出错: ' + error.message, 'error');
        });
}

// 显示通知消息
function showNotification(message, type = 'info') {
    // 创建通知元素
    const notification = document.createElement('div');
    notification.className = 'fixed top-4 right-4 z-50 p-4 rounded-lg shadow-lg transform transition-all duration-300';

    // 根据类型设置样式
    switch(type) {
        case 'success':
            notification.className += ' bg-green-500 text-white';
            break;
        case 'error':
            notification.className += ' bg-red-500 text-white';
            break;
        case 'warning':
            notification.className += ' bg-yellow-500 text-white';
            break;
        default:
            notification.className += ' bg-blue-500 text-white';
    }

    notification.innerHTML = `
        <div class="flex items-center">
            <i class="fas ${type === 'success' ? 'fa-check-circle' : type === 'error' ? 'fa-exclamation-circle' : 'fa-info-circle'} mr-2"></i>
            <span>${message}</span>
            <button onclick="this.parentElement.parentElement.remove()" class="ml-4 text-white hover:text-gray-200">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `;

    // 添加到页面
    document.body.appendChild(notification);

    // 3秒后自动消失
    setTimeout(() => {
        if (notification.parentElement) {
            notification.remove();
        }
    }, 3000);
}




// ... existing code ...

// 确保只定义一次这些函数
if (typeof toggleAudioForm !== 'function') {
    function toggleAudioForm(toolName) {
        const form = document.getElementById('form_' + toolName);
        if (form) {
            form.classList.toggle('hidden');
        }
    }
}

if (typeof hideAudioForm !== 'function') {
    function hideAudioForm(toolName) {
        const form = document.getElementById('form_' + toolName);
        if (form) {
            form.classList.add('hidden');
        }
    }
}

if (typeof toggleImageForm !== 'function') {
    function toggleImageForm(toolName) {
        const form = document.getElementById('form_' + toolName);
        if (form) {
            form.classList.toggle('hidden');
        }
    }
}

if (typeof hideImageForm !== 'function') {
    function hideImageForm(toolName) {
        const form = document.getElementById('form_' + toolName);
        if (form) {
            form.classList.add('hidden');
        }
    }
}

if (typeof toggleLyricForm !== 'function') {
    function toggleLyricForm(toolName) {
        const form = document.getElementById('form_' + toolName);
        if (form) {
            form.classList.toggle('hidden');
        }
    }
}

if (typeof hideLyricForm !== 'function') {
    function hideLyricForm(toolName) {
        const form = document.getElementById('form_' + toolName);
        if (form) {
            form.classList.add('hidden');
        }
    }
}

// ... existing code ...

function executeTool(toolName) {
    fetch('/api/execute', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({toolName: toolName})
    })
        .then(response => response.json())
        .then(data => {
            console.log('Tool execution initiated:', data);
        })
        .catch(error => {
            console.error('Error executing tool:', error);
        });
}

function closeModal() {
    const overlay = document.getElementById('modalOverlay');
    if (overlay) {
        overlay.remove();
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}



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

// 保存风格设置
function saveStyleSetting() {
    const selectElement = document.getElementById('styleTemplateSelect');
    console.log(selectElement.value)
    const selectedTemplateId = selectElement.value;

    // 确保选择了一个有效的模板
    if (!selectedTemplateId) {
        alert('请选择一个风格模板');
        return;
    }

    // 将设置保存到localStorage
    localStorage.setItem('selectedStyleTemplateId', selectedTemplateId);

    // 发送请求到后端保存设置
    fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            setting_type: 'style_template',
            setting_value: selectedTemplateId
        })
    })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert('风格设置已保存');
            } else {
                alert('保存设置失败: ' + data.message);
            }
        })
        .catch(error => {
            console.error('保存风格设置失败:', error);
            alert('保存设置失败: ' + error.message);
        });
}

// 重置设置
function resetSettings() {
    if (confirm('确定要重置所有设置吗？')) {
        // 重置表单为默认值
        document.getElementById('imageWidth').value = 512;
        document.getElementById('imageHeight').value = 896;
        document.getElementById('imageQuality').value = 'medium';
        document.getElementById('threadCount').value = 2;

        // 从localStorage清除设置
        localStorage.removeItem('imageWidth');
        localStorage.removeItem('imageHeight');
        localStorage.removeItem('imageQuality');
        localStorage.removeItem('threadCount');
        localStorage.removeItem('selectedStyleTemplateId');

        alert('设置已重置为默认值');
    }
}



// 初始化 - 加载工具列表
window.onload = function () {
    // 在初始状态下加载工具列表
    if (document.querySelector('.nav-tab.active').textContent.includes('MCP 工具')) {
        loadToolsList();
    }

    // 加载风格模板选择器
    loadStyleTemplates();

    // 加载保存的设置
    loadSavedSettings();
};

