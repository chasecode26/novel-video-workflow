// 文件管理相关功能函数
let currentDirectory = 'input';

function changeDirectory(dir) {
    currentDirectory = dir.startsWith('./') ? dir.substring(2) : dir;
    loadFileList(currentDirectory);
}

function loadFileList(dir) {
    let pathToUse = dir;
    if (!dir.startsWith('./')) {
        pathToUse = './' + dir;
    }

    fetch('/api/files/list?dir=' + encodeURIComponent(pathToUse))
        .then(response => response.json())
        .then(data => {
            const fileListDiv = document.getElementById('fileList');
            if (!fileListDiv) return;

            fileListDiv.innerHTML = '';

            if (!Array.isArray(data.files) || data.files.length === 0) {
                fileListDiv.innerHTML = '<div class="text-center py-16 text-2xl text-gray-300"><i class="fas fa-inbox text-6xl mb-6 opacity-50"></i><p>目录为空</p></div>';
                return;
            }

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
            data.files.forEach(file => {
                const row = document.createElement('tr');
                row.className = 'hover:bg-white hover:bg-opacity-10 transition-colors duration-150';

                const fileTypeDisplay = file.isDir ? '📁' : getFileIconByType(file.type);
                const fileSizeDisplay = file.isDir ? '-' : formatFileSize(file.size);
                let previewButton = '';
                if (!file.isDir && ['text', 'json', 'yaml', 'yml', 'xml', 'csv', 'log', 'md', 'image', 'audio'].includes(file.type)) {
                    previewButton = '<button onclick="previewFile(\'' + file.name + '\')" class="text-blue-300 hover:text-blue-100 mr-5 text-lg transition-colors duration-150"><i class="fas fa-eye mr-2"></i>预览</button>';
                }
                const modTime = new Date(file.modTime).toLocaleString();

                row.innerHTML = "<td class=\"px-6 py-6 whitespace-nowrap\">\n                                <div class=\"flex items-center\">\n                                    <div class=\"mr-5 text-2xl opacity-80\">" + fileTypeDisplay + "</div>\n                                    <div class=\"text-xl font-medium text-gray-200\">\n                                        <span onclick=\"clickFileOrDir('" + file.name + "', " + file.isDir + ")\" class=\"cursor-pointer hover:text-blue-300 transition-colors duration-150\">" + file.name + "</span>\n                                    </div>\n                                </div>\n                            </td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl text-gray-300\">" + file.type + "</td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl text-gray-300\">" + fileSizeDisplay + "</td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl text-gray-300\">" + modTime + "</td>\n                            <td class=\"px-6 py-6 whitespace-nowrap text-xl font-medium\">\n                                " + previewButton + "\n                                <button onclick=\"deleteFile('" + file.name + "', " + file.isDir + ")\" class=\"text-red-400 hover:text-red-200 transition-colors duration-150\"><i class=\"fas fa-trash mr-2\"></i>删除</button>\n                            </td>\n                        ";
                tbody.appendChild(row);
            });

            fileListDiv.appendChild(table);
        })
        .catch(error => {
            console.error('Error loading file list:', error);
            const fileListDiv = document.getElementById('fileList');
            if (fileListDiv) {
                fileListDiv.innerHTML = '<div class="text-red-400 p-8 text-center text-xl"><i class="fas fa-exclamation-triangle mr-3"></i>加载文件列表失败: ' + error.message + '</div>';
            }
        });
}

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

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function clickFileOrDir(name, isDir) {
    if (isDir) {
        let newPath = currentDirectory;
        if (newPath.endsWith('/')) {
            newPath += name;
        } else {
            newPath += '/' + name;
        }
        changeDirectory('./' + newPath);
        return;
    }

    previewFile(name);
}

function previewFile(filename) {
    const fullPath = './' + currentDirectory + '/' + filename;
    const ext = filename.split('.').pop().toLowerCase();
    const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'];
    const audioExtensions = ['mp3', 'wav', 'ogg', 'm4a', 'flac'];
    const textExtensions = ['txt', 'json', 'yaml', 'yml', 'xml', 'csv', 'log', 'md'];
    const subtitleExtensions = ['srt', 'ass', 'vtt'];

    if (imageExtensions.includes(ext)) {
        showImagePreview(convertToStaticPath(fullPath), filename);
        return;
    }
    if (audioExtensions.includes(ext)) {
        showAudioPreview(convertToStaticPath(fullPath), filename);
        return;
    }
    if (subtitleExtensions.includes(ext) || textExtensions.includes(ext)) {
        fetch('/api/files/content?path=' + encodeURIComponent(fullPath))
            .then(response => response.text())
            .then(content => {
                showTextPreview(content, filename);
            })
            .catch(error => {
                console.error('Error previewing file:', error);
                alert('无法预览文件: ' + error.message);
            });
        return;
    }

    fetch('/api/files/content?path=' + encodeURIComponent(fullPath))
        .then(response => {
            if (!response.ok) {
                throw new Error('HTTP error! status: ' + response.status);
            }
            return response.text();
        })
        .then(content => {
            showTextPreview(content, filename);
        })
        .catch(error => {
            console.error('Error previewing file:', error);
            alert('无法预览文件: ' + error.message);
        });
}

function convertToStaticPath(relativePath) {
    if (relativePath.startsWith('./input/')) {
        return '/files/input/' + relativePath.substring(9);
    }
    if (relativePath.startsWith('./output/')) {
        return '/files/output/' + relativePath.substring(9);
    }
    return relativePath;
}

function showTextPreview(content, filename) {
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

function showImagePreview(filePath, filename) {
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

function showAudioPreview(filePath, filename) {
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

function deleteFile(filename, isDir) {
    const confirmMessage = '确定要删除' + (isDir ? '目录' : '文件') + ' "' + filename + '" 吗？此操作不可撤销。';
    if (!confirm(confirmMessage)) {
        return;
    }

    const fullPath = './' + currentDirectory + '/' + filename;
    fetch('/api/files/delete?path=' + encodeURIComponent(fullPath), {
        method: 'DELETE'
    })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert((isDir ? '目录' : '文件') + ' 已成功删除');
                loadFileList(currentDirectory);
                return;
            }
            alert('删除失败: ' + data.message);
        })
        .catch(error => {
            console.error('Error deleting file:', error);
            alert('删除失败: ' + error.message);
        });
}

function initDragAndDrop() {
    const dropArea = document.getElementById('folderUploadArea');
    if (!dropArea) {
        return;
    }

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
        if (typeof handleFolderUpload === 'function') {
            handleFolderUpload(files);
        }
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initDragAndDrop);
} else {
    initDragAndDrop();
}

function loadFileManager() {
    loadFileList('input');
}
