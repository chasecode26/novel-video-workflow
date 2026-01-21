// 设置相关功能函数

// 加载风格模板
function loadStyleTemplates() {
    fetch('/api/prompt-templates')
        .then(response => response.json())
        .then(templates => {
            const selectElement = document.getElementById('styleTemplateSelect');
            if (!selectElement) return;
            
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
}

// 保存风格设置
function saveStyleSetting() {
    const selectElement = document.getElementById('styleTemplateSelect');
    if (!selectElement) return;
    
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
        const imageWidthEl = document.getElementById('imageWidth');
        const imageHeightEl = document.getElementById('imageHeight');
        const imageQualityEl = document.getElementById('imageQuality');
        const threadCountEl = document.getElementById('threadCount');
        
        if (imageWidthEl) imageWidthEl.value = 512;
        if (imageHeightEl) imageHeightEl.value = 896;
        if (imageQualityEl) imageQualityEl.value = 'medium';
        if (threadCountEl) threadCountEl.value = 2;
        
        // 从localStorage清除设置
        localStorage.removeItem('imageWidth');
        localStorage.removeItem('imageHeight');
        localStorage.removeItem('imageQuality');
        localStorage.removeItem('threadCount');
        localStorage.removeItem('selectedStyleTemplateId');
        
        alert('设置已重置为默认值');
    }
}

// 加载模板
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

// 分享章节功能
let currentChapterId = null; // 全局变量，存储当前章节ID

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

// 取消分享功能
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

