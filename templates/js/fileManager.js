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

// 处理文件上传
function handleFileUpload(files) {
    if (files.length === 0) return;

    // 处理多个文件上传，保持目录结构
    uploadMultipleFiles(Array.from(files), './' + currentDirectory);
}

function handleFolderUpload(files) {
    if (files.length === 0) return;

    // 上传文件夹中的所有文件，保持目录结构
    uploadMultipleFiles(Array.from(files), './input');
}

function uploadMultipleFiles(files, baseDir) {
    if (files.length === 0) return;

    // 显示上传进度
    const uploadStatus = document.getElementById('uploadStatus');
    if (uploadStatus) {
        uploadStatus.innerHTML = '开始上传 ' + files.length + ' 个文件...';
    }

    // 递归上传文件数组中的每个文件
    const uploadNext = (index) => {
        if (index >= files.length) {
            // 所有文件上传完成后刷新文件列表
            loadFileList(currentDirectory);
            if (uploadStatus) {
                uploadStatus.innerHTML = '所有文件上传完成 (' + files.length + ' 个文件)';
            }
            return;
        }

        const file = files[index];

        // 创建FormData对象
        const formData = new FormData();
        formData.append('file', file);

        // 处理文件路径，保持目录结构
        let filePath = file.webkitRelativePath || file.name;
        if (filePath.startsWith('./')) {
            filePath = filePath.substring(2);
        }

        // 计算完整的目标路径
        let fullDestPath = baseDir;
        if (fullDestPath.endsWith('/')) {
            fullDestPath += filePath;
        } else {
            fullDestPath += '/' + filePath;
        }

        // 获取目录部分，用于创建目录
        let destDir = fullDestPath.substring(0, fullDestPath.lastIndexOf('/'));
        if (destDir === '') {
            destDir = baseDir;
        }

        formData.append('dir', destDir);

        // 显示上传状态
        if (uploadStatus) {
            uploadStatus.innerHTML = '上传中: ' + filePath + ' (' + (index + 1) + '/' + files.length + ')';
        }

        fetch('/api/files/upload', {
            method: 'POST',
            body: formData
        })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    console.log('文件上传成功: ' + data.filename);
                } else {
                    console.error('上传失败: ' + data.message);
                }

                // 继续上传下一个文件
                uploadNext(index + 1);
            })
            .catch(function (error) {
                console.error('Error uploading file:', error);
                if (uploadStatus) {
                    uploadStatus.innerHTML = '上传失败: ' + error.message;
                }

                // 即使出错也继续上传下一个文件
                uploadNext(index + 1);
            });
    };

    uploadNext(0);
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

// 当切换到文件管理标签时加载文件列表
function loadFileManager() {
    // 使用input作为默认目录
    loadFileList('input');
}

// 处理文件上传
function handleFileUpload(files) {
    if (files.length === 0) return;

    // 处理多个文件上传，保持目录结构
    uploadMultipleFiles(Array.from(files), './' + currentDirectory);
}

function handleFolderUpload(files) {
    if (files.length === 0) return;

    // 上传文件夹中的所有文件，保持目录结构
    uploadMultipleFiles(Array.from(files), './input');
}

function uploadMultipleFiles(files, baseDir) {
    if (files.length === 0) return;

    // 显示上传进度
    const uploadStatus = document.getElementById('uploadStatus');
    if (uploadStatus) {
        uploadStatus.innerHTML = '开始上传 ' + files.length + ' 个文件...';
    }

    // 递归上传文件数组中的每个文件
    const uploadNext = (index) => {
        if (index >= files.length) {
            // 所有文件上传完成后刷新文件列表
            loadFileList(currentDirectory);
            if (uploadStatus) {
                uploadStatus.innerHTML = '所有文件上传完成 (' + files.length + ' 个文件)';
            }
            return;
        }

        const file = files[index];

        // 创建FormData对象
        const formData = new FormData();
        formData.append('file', file);

        // 处理文件路径，保持目录结构
        let filePath = file.webkitRelativePath || file.name;
        if (filePath.startsWith('./')) {
            filePath = filePath.substring(2);
        }

        // 计算完整的目标路径
        let fullDestPath = baseDir;
        if (fullDestPath.endsWith('/')) {
            fullDestPath += filePath;
        } else {
            fullDestPath += '/' + filePath;
        }

        // 获取目录部分，用于创建目录
        let destDir = fullDestPath.substring(0, fullDestPath.lastIndexOf('/'));
        if (destDir === '') {
            destDir = baseDir;
        }

        formData.append('dir', destDir);

        // 显示上传状态
        if (uploadStatus) {
            uploadStatus.innerHTML = '上传中: ' + filePath + ' (' + (index + 1) + '/' + files.length + ')';
        }

        fetch('/api/files/upload', {
            method: 'POST',
            body: formData
        })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    console.log('文件上传成功: ' + data.filename);
                } else {
                    console.error('上传失败: ' + data.message);
                }

                // 继续上传下一个文件
                uploadNext(index + 1);
            })
            .catch(function (error) {
                console.error('Error uploading file:', error);
                if (uploadStatus) {
                    uploadStatus.innerHTML = '上传失败: ' + error.message;
                }

                // 即使出错也继续上传下一个文件
                uploadNext(index + 1);
            });
    };

    uploadNext(0);
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
                'default': 'fas fa-cog'
            };

            tools.forEach(function (tool) {
                const toolCard = document.createElement('div');
                toolCard.className = 'glass-effect rounded-2xl p-6 border border-white border-opacity-20 card-hover future-glow';

                // 获取对应工具的图标
                const iconClass = toolIcons[tool.name] || toolIcons['default'];

                // 为generate_indextts2_audio工具添加特殊处理
                let cardContent = '';
                if (tool.name === 'generate_indextts2_audio') {
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

// 显示音频生成表单
function toggleAudioForm(toolName) {
    const form = document.getElementById('form_' + toolName);
    if (form) {
        form.classList.toggle('hidden');
    }
}

// 隐藏音频生成表单
function hideAudioForm(toolName) {
    const form = document.getElementById('form_' + toolName);
    if (form) {
        form.classList.add('hidden');
    }
}

// 显示图像生成表单
function toggleImageForm(toolName) {
    const form = document.getElementById('form_' + toolName);
    if (form) {
        form.classList.toggle('hidden');
    }
}

// 隐藏图像生成表单
function hideImageForm(toolName) {
    const form = document.getElementById('form_' + toolName);
    if (form) {
        form.classList.add('hidden');
    }
}

// 执行音频生成工具
function executeAudioTool(toolName) {
    const textInput = document.getElementById('textInput_' + toolName).value;
    const outputDir = document.getElementById('outputDir_' + toolName).value;

    if (!textInput || textInput.trim() === '') {
        alert('请输入要转换为语音的文本');
        return;
    }

    // 生成输出文件路径
    const timestamp = new Date().getTime();
    const outputFile = outputDir + '/audio_' + timestamp + '.wav';

    const params = {
        toolName: toolName,
        text: textInput,
        output_file: outputFile
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
            console.log('Audio tool execution initiated:', data);
            // 隐藏表单
            hideAudioForm(toolName);
        })
        .catch(error => {
            console.error('Error executing audio tool:', error);
        });
}

// 执行图像生成工具
function executeImageTool(toolName) {
    const chapterText = document.getElementById('chapterText_' + toolName).value;
    const outputDir = document.getElementById('outputDir_' + toolName).value;
    const imageWidth = parseInt(document.getElementById('imageWidth_' + toolName).value);
    const imageHeight = parseInt(document.getElementById('imageHeight_' + toolName).value);

    if (!chapterText || chapterText.trim() === '') {
        alert('请输入要生成图像的章节文本');
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
        chapter_text: chapterText,
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
            console.log('Image tool execution initiated:', data);
            // 隐藏表单
            hideImageForm(toolName);
        })
        .catch(error => {
            console.error('Error executing image tool:', error);
        });
}

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

function loadTemplates() {
    fetch('/api/templates')
        .then(response => response.json())
        .then(data => {
            const select = document.getElementById('styleTemplateSelect');
            data.templates.forEach(template => {
                const option = document.createElement('option');
                option.value = template.ID;
                option.textContent = template.name;
                select.appendChild(option);
            });
        })
        .catch(error => {
            console.error('Error loading templates:', error);
        });
}


function handleDragOver(e) {
    e.preventDefault();
    e.stopPropagation();
    document.getElementById('uploadArea').classList.add('border-blue-400', 'bg-blue-400', 'bg-opacity-10');
}

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

// 保存通用设置
function saveGeneralSettings() {
    const imageWidth = document.getElementById('imageWidth').value;
    const imageHeight = document.getElementById('imageHeight').value;
    const imageQuality = document.getElementById('imageQuality').value;
    const threadCount = document.getElementById('threadCount').value;

    // 将设置保存到localStorage
    localStorage.setItem('imageWidth', imageWidth);
    localStorage.setItem('imageHeight', imageHeight);
    localStorage.setItem('imageQuality', imageQuality);
    localStorage.setItem('threadCount', threadCount);

    // 发送请求到后端保存设置
    fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            setting_type: 'general',
            setting_value: {
                image_width: parseInt(imageWidth),
                image_height: parseInt(imageHeight),
                image_quality: imageQuality,
                thread_count: parseInt(threadCount)
            }
        })
    })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert('通用设置已保存');
            } else {
                alert('保存设置失败: ' + data.message);
            }
        })
        .catch(error => {
            console.error('保存通用设置失败:', error);
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

// 生成分享链接
function shareChapter() {
    if (!currentChapterId) {
        alert('请先选择一个章节');
        return;
    }

    const password = document.getElementById('sharePassword').value;
    if (!password) {
        alert('请输入分享密码');
        return;
    }

    fetch('/api/chapters/' + currentChapterId + '/share', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            password: password
        })
    })
        .then(response => response.json())
        .then(data => {
            document.getElementById('shareResult').classList.remove('hidden');
            document.getElementById('shareLink').textContent = window.location.origin + '/chapters/share/public/' + data.share_token;
            document.getElementById('sharePassword').value = '';
        })
        .catch(error => {
            console.error('生成分享链接失败:', error);
            alert('生成分享链接失败: ' + error.message);
        });
}

// 取消分享
function unshareChapter() {
    if (!currentChapterId) {
        alert('请先选择一个章节');
        return;
    }

    fetch('/api/chapters/' + currentChapterId + '/unshare', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({})
    })
        .then(response => response.json())
        .then(data => {
            alert('分享已取消');
            document.getElementById('shareResult').classList.add('hidden');
            document.getElementById('shareLink').textContent = '';
        })
        .catch(error => {
            console.error('取消分享失败:', error);
            alert('取消分享失败: ' + error.message);
        });
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



// 加载保存的设置
function loadSavedSettings() {
    // 加载风格模板选择
    const savedTemplateId = localStorage.getItem('selectedStyleTemplateId');
    if (savedTemplateId) {
        const selectElement = document.getElementById('styleTemplateSelect');
        if (selectElement) {
            selectElement.value = savedTemplateId;
        }
    }

    // 加载通用设置
    const savedImageWidth = localStorage.getItem('imageWidth');
    const savedImageHeight = localStorage.getItem('imageHeight');
    const savedImageQuality = localStorage.getItem('imageQuality');
    const savedThreadCount = localStorage.getItem('threadCount');

    if (savedImageWidth) {
        document.getElementById('imageWidth').value = savedImageWidth;
    }
    if (savedImageHeight) {
        document.getElementById('imageHeight').value = savedImageHeight;
    }
    if (savedImageQuality) {
        document.getElementById('imageQuality').value = savedImageQuality;
    }
    if (savedThreadCount) {
        document.getElementById('threadCount').value = savedThreadCount;
    }
};

// 加载风格模板
function loadStyleTemplates() {
    fetch('/api/prompt-templates')
        .then(response => response.json())
        .then(templates => {
            const selectElement = document.getElementById('styleTemplateSelect');
            // 清空现有选项（除了默认选项）
            selectElement.innerHTML = '<option value="">默认风格</option>';

            // 添加模板选项
            templates.forEach(template => {
                const option = document.createElement('option');
                option.value = template.ID;
                option.textContent = template.name + ' - ' + template.description;
                selectElement.appendChild(option);
            });
        })
        .catch(error => {
            console.error('加载风格模板失败:', error);
            // 即使加载失败也不显示错误，只是使用默认选项
        });
};