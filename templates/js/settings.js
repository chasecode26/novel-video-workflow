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

async function loadServerSettings() {
    const response = await fetch('/api/settings');
    const payload = await response.json();
    if (!response.ok || payload.status !== 'success' || !payload.data) {
        throw new Error(payload.message || '加载配置失败');
    }

    const data = payload.data;
    const general = data.general || {};
    const tts = data.tts || {};
    const subtitle = data.subtitle || {};
    const project = data.project || {};
    const ollama = data.ollama || {};

    const imageWidthEl = document.getElementById('imageWidth');
    const imageHeightEl = document.getElementById('imageHeight');
    const ollamaApiUrlEl = document.getElementById('ollamaApiUrl');
    const ollamaModelEl = document.getElementById('ollamaModel');
    const ollamaTimeoutEl = document.getElementById('ollamaTimeoutSeconds');

    if (imageWidthEl && general.image_width) imageWidthEl.value = general.image_width;
    if (imageHeightEl && general.image_height) imageHeightEl.value = general.image_height;
    const assignValue = (id, value) => {
        const element = document.getElementById(id);
        if (element && value !== undefined && value !== null && value !== '') {
            element.value = value;
        }
    };
    const assignChecked = (id, value) => {
        const element = document.getElementById(id);
        if (element) {
            element.checked = !!value;
        }
    };
    assignValue('ttsProvider', tts.provider);
    assignValue('ttsApiUrl', tts.api_url);
    assignValue('referenceAudio', tts.reference_audio);
    assignValue('ttsVoiceModel', tts.voice_model);
    assignValue('ttsTimeoutSeconds', tts.timeout_seconds);
    assignValue('ttsMaxRetries', tts.max_retries);
    assignValue('ttsSampleRate', tts.sample_rate);
    assignValue('ttsPythonPath', tts.python_path);
    assignValue('ttsIndexPath', tts.indextts_path);
    assignValue('subtitleProvider', subtitle.provider);
    assignValue('subtitleStyle', subtitle.style);
    assignValue('subtitleFontName', subtitle.font_name);
    assignValue('subtitleFontSize', subtitle.font_size);
    assignValue('subtitleExecutablePath', subtitle.executable_path);
    assignValue('subtitleScriptPath', subtitle.script_path);
    assignChecked('subtitleUseAutomation', subtitle.use_automation);
    assignValue('projectProvider', project.provider);
    if (ollamaApiUrlEl && ollama.api_url) ollamaApiUrlEl.value = ollama.api_url;
    if (ollamaModelEl && ollama.model) ollamaModelEl.value = ollama.model;
    if (ollamaTimeoutEl && ollama.timeout_seconds) ollamaTimeoutEl.value = ollama.timeout_seconds;
}

function saveTtsSettings() {
    fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            setting_type: 'tts',
            setting_value: {
                provider: document.getElementById('ttsProvider')?.value?.trim(),
                api_url: document.getElementById('ttsApiUrl')?.value?.trim(),
                reference_audio: document.getElementById('referenceAudio')?.value?.trim(),
                voice_model: document.getElementById('ttsVoiceModel')?.value?.trim(),
                timeout_seconds: parseInt(document.getElementById('ttsTimeoutSeconds')?.value || '300', 10),
                max_retries: parseInt(document.getElementById('ttsMaxRetries')?.value || '3', 10),
                sample_rate: parseInt(document.getElementById('ttsSampleRate')?.value || '24000', 10),
                python_path: document.getElementById('ttsPythonPath')?.value?.trim(),
                indextts_path: document.getElementById('ttsIndexPath')?.value?.trim()
            }
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            alert('TTS 配置已保存');
        } else {
            alert('保存 TTS 配置失败: ' + data.message);
        }
    })
    .catch(error => {
        console.error('保存 TTS 配置失败:', error);
        alert('保存 TTS 配置失败: ' + error.message);
    });
}

function saveSubtitleSettings() {
    fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            setting_type: 'subtitle',
            setting_value: {
                provider: document.getElementById('subtitleProvider')?.value?.trim(),
                style: document.getElementById('subtitleStyle')?.value?.trim(),
                font_name: document.getElementById('subtitleFontName')?.value?.trim(),
                font_size: parseInt(document.getElementById('subtitleFontSize')?.value || '48', 10),
                executable_path: document.getElementById('subtitleExecutablePath')?.value?.trim(),
                script_path: document.getElementById('subtitleScriptPath')?.value?.trim(),
                use_automation: !!document.getElementById('subtitleUseAutomation')?.checked
            }
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            alert('字幕配置已保存');
        } else {
            alert('保存字幕配置失败: ' + data.message);
        }
    })
    .catch(error => {
        console.error('保存字幕配置失败:', error);
        alert('保存字幕配置失败: ' + error.message);
    });
}

function saveProjectSettings() {
    fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            setting_type: 'project',
            setting_value: {
                provider: document.getElementById('projectProvider')?.value?.trim()
            }
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            alert('成片配置已保存');
        } else {
            alert('保存成片配置失败: ' + data.message);
        }
    })
    .catch(error => {
        console.error('保存成片配置失败:', error);
        alert('保存成片配置失败: ' + error.message);
    });
}

function saveOllamaSettings() {
    const apiUrl = document.getElementById('ollamaApiUrl')?.value?.trim();
    const model = document.getElementById('ollamaModel')?.value?.trim();
    const timeoutSeconds = parseInt(document.getElementById('ollamaTimeoutSeconds')?.value || '120', 10);

    if (!apiUrl || !model) {
        alert('请填写完整的 Ollama 地址和模型名');
        return;
    }

    fetch('/api/settings', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            setting_type: 'ollama',
            setting_value: {
                api_url: apiUrl,
                model: model,
                timeout_seconds: timeoutSeconds
            }
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            alert('Ollama 配置已保存');
        } else {
            alert('保存 Ollama 配置失败: ' + data.message);
        }
    })
    .catch(error => {
        console.error('保存 Ollama 配置失败:', error);
        alert('保存 Ollama 配置失败: ' + error.message);
    });
}

const originalLoadSavedSettings = loadSavedSettings;
loadSavedSettings = function() {
    if (typeof originalLoadSavedSettings === 'function') {
        originalLoadSavedSettings();
    }
    loadServerSettings().catch(error => {
        console.error('加载服务器配置失败:', error);
    });
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

