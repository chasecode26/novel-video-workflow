// 上传功能相关函数

// 处理文件夹上传
function handleFolderUpload(files) {
    if (files.length === 0) return;
    
    // 上传文件夹中的所有文件，保持目录结构
    uploadMultipleFiles(Array.from(files), './input');
}

// 上传多个文件
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
            if (typeof loadFileList !== 'undefined' && typeof currentDirectory !== 'undefined') {
                loadFileList(currentDirectory);
            }
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
        .catch(function(error) {
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

// 上传功能相关
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