// 章节场景管理相关函数
let currentSelectedChapterId = null;

// 加载章节列表
function loadChaptersList() {
    fetch('/api/chapters')
        .then(response => response.json())
        .then(data => {
            const chaptersListDiv = document.getElementById('chaptersList');
            if (!chaptersListDiv) return;
            
            chaptersListDiv.innerHTML = '';

            if (data.chapters && data.chapters.length > 0) {
                data.chapters.forEach(chapter => {
                    const chapterCard = document.createElement('div');
                    chapterCard.className = 'mb-2 cursor-pointer rounded-lg p-3 hover:bg-white hover:bg-opacity-20 transition-colors duration-150';
                    // 处理标题显示，移除可能的空白字符并确保正确显示
                    const displayTitle = (chapter.Title || '无标题').trim();
                    chapterCard.innerHTML = '<div class="flex justify-between items-center">' +
                        '<div>' +
                            '<h4 class="font-medium text-white">' + displayTitle + '</h4>' +
                            '<p class="text-xs text-gray-400">章节ID: ' + chapter.ID + '</p>' +
                        '</div>' +
                        '<button onclick="selectChapter(' + chapter.ID + ')" class="text-blue-300 hover:text-blue-100 text-sm">' +
                            '<i class="fas fa-edit"></i>' +
                        '</button>' +
                    '</div>';
                    chaptersListDiv.appendChild(chapterCard);
                });
            } else {
                chaptersListDiv.innerHTML = '<p class="text-gray-400 text-center py-4">暂无章节</p>';
            }
        })
        .catch(error => {
            console.error('加载章节列表失败:', error);
            const chaptersListDiv = document.getElementById('chaptersList');
            if (chaptersListDiv) {
                chaptersListDiv.innerHTML = '<p class="text-red-400 text-center py-4">加载章节列表失败</p>';
            }
        });
}

// 选择章节
function selectChapter(chapterId) {
    currentSelectedChapterId = chapterId;
    loadScenesList(chapterId);
    
    // 更新选中的章节显示
    fetch('/api/chapters/' + chapterId)
        .then(response => response.json())
        .then(function(chapter) {
            // 处理章节标题显示，移除可能的空白字符并确保正确显示
            const displayTitle = (chapter.Title || '无标题').trim();
            const selectedChapterTitle = document.getElementById('selectedChapterTitle');
            if (selectedChapterTitle) {
                selectedChapterTitle.textContent = ' - ' + displayTitle;
            }
        })
        .catch(error => {
            console.error('获取章节信息失败:', error);
            const selectedChapterTitle = document.getElementById('selectedChapterTitle');
            if (selectedChapterTitle) {
                selectedChapterTitle.textContent = ' - 加载失败';
            }
        });
}

// 加载场景列表
function loadScenesList(chapterId) {
    fetch('/api/scenes?chapter_id=' + chapterId)
        .then(response => response.json())
        .then(data => {
            const scenesListDiv = document.getElementById('scenesList');
            if (!scenesListDiv) return;
            
            scenesListDiv.innerHTML = '';

            if (data.scenes && data.scenes.length > 0) {
                data.scenes.forEach(scene => {
                    const sceneCard = document.createElement('div');
                    sceneCard.className = 'mb-3 rounded-lg p-4 bg-black bg-opacity-20 border border-white border-opacity-20';
                    
                    // 获取场景缩略图
                    let thumbnailHtml = '<div class="w-16 h-16 bg-gray-700 rounded-lg flex items-center justify-center text-gray-400 text-xs">无图</div>';
                    if (scene.ImagePath) {
                        const imagePath = scene.ImagePath.startsWith('/') ? scene.ImagePath : '/' + scene.ImagePath;
                        thumbnailHtml = '<img src="' + imagePath + '" alt="场景图片" class="w-16 h-16 object-cover rounded-lg" onerror="this.onerror=null; this.parentElement.innerHTML=\'<div class="w-16 h-16 bg-gray-700 rounded-lg flex items-center justify-center text-gray-400 text-xs">无图</div>\';">';
                    }
                    
                    // 处理Ollama响应显示，移除可能的空白字符并确保正确显示
                    const displayOllamaResponse = (scene.OllamaResponse || '无内容').trim();
                    sceneCard.innerHTML = '<div class="flex gap-4">' +
                        '<div class="flex-shrink-0">' +
                            thumbnailHtml +
                        '</div>' +
                        '<div class="flex-grow">' +
                            '<div class="flex justify-between">' +
                                '<h4 class="font-medium text-white">场景 ' + scene.ID + '</h4>' +
                                '<div class="flex gap-2">' +
                                    '<button onclick="editScene(' + scene.ID + ')" class="text-blue-300 hover:text-blue-100 text-sm" title="编辑">' +
                                        '<i class="fas fa-edit"></i>' +
                                    '</button>' +
                                    '<button onclick="retryScene(' + scene.ID + ')" class="text-green-300 hover:text-green-100 text-sm" title="重试">' +
                                        '<i class="fas fa-sync-alt"></i>' +
                                    '</button>' +
                                    '<button onclick="deleteScene(' + scene.ID + ')" class="text-red-300 hover:text-red-100 text-sm" title="删除">' +
                                        '<i class="fas fa-trash"></i>' +
                                    '</button>' +
                                '</div>' +
                            '</div>' +
                            '<p class="text-sm text-gray-300 mt-1 truncate">' + displayOllamaResponse + '</p>' +
                            '<div class="mt-2 text-xs text-gray-400">' +
                                '<p>重试次数: ' + (scene.RetryCount || 0) + '</p>' +
                                '<p>状态: ' + (scene.Status || '未知') + '</p>' +
                            '</div>' +
                        '</div>' +
                    '</div>';
                    scenesListDiv.appendChild(sceneCard);
                });
            } else {
                scenesListDiv.innerHTML = '<p class="text-gray-400 text-center py-4">该章节下暂无场景</p>';
            }
        })
        .catch(error => {
            console.error('加载场景列表失败:', error);
            const scenesListDiv = document.getElementById('scenesList');
            if (scenesListDiv) {
                scenesListDiv.innerHTML = '<p class="text-red-400 text-center py-4">加载场景列表失败</p>';
            }
        });
}

// 编辑章节
function editChapter(chapterId) {
    fetch('/api/chapters/' + chapterId)
        .then(response => response.json())
        .then(chapter => {
            // 创建模态框编辑章节
            const overlay = document.createElement('div');
            overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
            overlay.id = 'modalOverlay';

            const modal = document.createElement('div');
            modal.className = 'glass-effect rounded-2xl w-full max-w-3xl max-h-[90vh] flex flex-col border border-white border-opacity-30';

            const header = document.createElement('div');
            header.className = 'flex justify-between items-center p-6 border-b border-white border-opacity-30 rounded-t-2xl';
            header.innerHTML = '<h3 class="text-xl font-bold text-white">编辑章节</h3>' +
                '<button onclick="closeModal()" class="text-gray-300 hover:text-white text-3xl leading-none">' +
                    '<i class="fas fa-times"></i>' +
                '</button>';

            const contentDiv = document.createElement('div');
            contentDiv.className = 'p-6 flex-grow overflow-auto';
            contentDiv.innerHTML = '<div class="space-y-4">' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">章节标题</label>' +
                    '<input type="text" id="chapterTitle" value="' + (chapter.Title || '') + '" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">章节内容</label>' +
                    '<textarea id="chapterContent" rows="6" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' + (chapter.Content || '') + '</textarea>' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">分割提示词</label>' +
                    '<textarea id="segmentationPrompt" rows="4" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' + (chapter.SegmentationPrompt || '') + '</textarea>' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">工作流参数</label>' +
                    '<textarea id="workflowParams" rows="6" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' + (chapter.WorkflowParams || '') + '</textarea>' +
                '</div>' +
            '</div>';

            const footer = document.createElement('div');
            footer.className = 'p-6 border-t border-white border-opacity-30 rounded-b-2xl flex justify-end gap-3';
            footer.innerHTML = '<button onclick="closeModal()" class="bg-gray-500 hover:bg-gray-600 text-white px-6 py-2 rounded-lg transition-colors">' +
                '取消' +
                '</button>' +
                '<button onclick="saveChapter(' + chapterId + ')" class="bg-blue-500 hover:bg-blue-600 text-white px-6 py-2 rounded-lg transition-colors">' +
                '保存' +
                '</button>';

            modal.appendChild(header);
            modal.appendChild(contentDiv);
            modal.appendChild(footer);
            overlay.appendChild(modal);
            document.body.appendChild(overlay);
        })
        .catch(error => {
            console.error('获取章节信息失败:', error);
            alert('获取章节信息失败: ' + error.message);
        });
}

// 保存章节
function saveChapter(chapterId) {
    const title = document.getElementById('chapterTitle').value;
    const content = document.getElementById('chapterContent').value;
    const segmentationPrompt = document.getElementById('segmentationPrompt').value;
    const workflowParams = document.getElementById('workflowParams').value;

    fetch('/api/chapters/' + chapterId, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            title: title,
            content: content,
            segmentation_prompt: segmentationPrompt,
            workflow_params: workflowParams
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            alert('章节已保存');
            closeModal();
            loadChaptersList(); // 刷新章节列表
            if (currentSelectedChapterId) {
                loadScenesList(currentSelectedChapterId); // 刷新场景列表
            }
        } else {
            alert('保存失败: ' + data.message);
        }
    })
    .catch(error => {
        console.error('保存章节失败:', error);
        alert('保存章节失败: ' + error.message);
    });
}

// 编辑场景
function editScene(sceneId) {
    fetch('/api/scenes/' + sceneId)
        .then(response => response.json())
        .then(scene => {
            // 创建模态框编辑场景
            const overlay = document.createElement('div');
            overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
            overlay.id = 'modalOverlay';

            const modal = document.createElement('div');
            modal.className = 'glass-effect rounded-2xl w-full max-w-3xl max-h-[90vh] flex flex-col border border-white border-opacity-30';

            const header = document.createElement('div');
            header.className = 'flex justify-between items-center p-6 border-b border-white border-opacity-30 rounded-t-2xl';
            header.innerHTML = '<h3 class="text-xl font-bold text-white">编辑场景</h3>' +
                '<button onclick="closeModal()" class="text-gray-300 hover:text-white text-3xl leading-none">' +
                    '<i class="fas fa-times"></i>' +
                '</button>';

            const contentDiv = document.createElement('div');
            contentDiv.className = 'p-6 flex-grow overflow-auto';
            contentDiv.innerHTML = '<div class="space-y-4">' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">Ollama请求</label>' +
                    '<textarea id="ollamaRequest" rows="4" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' + (scene.OllamaRequest || '') + '</textarea>' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">Ollama响应</label>' +
                    '<textarea id="ollamaResponse" rows="4" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' + (scene.OllamaResponse || '') + '</textarea>' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">绘图配置</label>' +
                    '<textarea id="drawThingsConfig" rows="6" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' + (scene.DrawThingsConfig || '') + '</textarea>' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">图像路径</label>' +
                    '<input type="text" id="imagePath" value="' + (scene.ImagePath || '') + '" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' +
                '</div>' +
                '<div>' +
                    '<label class="block text-sm font-medium text-gray-300 mb-2">状态</label>' +
                    '<input type="text" id="status" value="' + (scene.Status || '') + '" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">' +
                '</div>' +
            '</div>';

            const footer = document.createElement('div');
            footer.className = 'p-6 border-t border-white border-opacity-30 rounded-b-2xl flex justify-end gap-3';
            footer.innerHTML = '<button onclick="closeModal()" class="bg-gray-500 hover:bg-gray-600 text-white px-6 py-2 rounded-lg transition-colors">' +
                '取消' +
                '</button>' +
                '<button onclick="saveScene(' + sceneId + ')" class="bg-blue-500 hover:bg-blue-600 text-white px-6 py-2 rounded-lg transition-colors">' +
                '保存' +
                '</button>';

            modal.appendChild(header);
            modal.appendChild(contentDiv);
            modal.appendChild(footer);
            overlay.appendChild(modal);
            document.body.appendChild(overlay);
        })
        .catch(error => {
            console.error('获取场景信息失败:', error);
            alert('获取场景信息失败: ' + error.message);
        });
}

// 保存场景
function saveScene(sceneId) {
    const ollamaRequest = document.getElementById('ollamaRequest').value;
    const ollamaResponse = document.getElementById('ollamaResponse').value;
    const drawThingsConfig = document.getElementById('drawThingsConfig').value;
    const imagePath = document.getElementById('imagePath').value;
    const status = document.getElementById('status').value;

    fetch('/api/scenes/' + sceneId, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            ollama_request: ollamaRequest,
            ollama_response: ollamaResponse,
            draw_things_config: drawThingsConfig,
            image_path: imagePath,
            status: status
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            alert('场景已保存');
            closeModal();
            if (currentSelectedChapterId) {
                loadScenesList(currentSelectedChapterId); // 刷新场景列表
            }
        } else {
            alert('保存失败: ' + data.message);
        }
    })
    .catch(error => {
        console.error('保存场景失败:', error);
        alert('保存场景失败: ' + error.message);
    });
}

// 重试场景
function retryScene(sceneId) {
    if (confirm('确定要重试此场景吗？这将重新生成图像。')) {
        fetch('/api/scenes/' + sceneId + '/retry', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({})
        })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert('重试已启动');
                if (currentSelectedChapterId) {
                    loadScenesList(currentSelectedChapterId); // 刷新场景列表
                }
            } else {
                alert('重试失败: ' + data.message);
            }
        })
        .catch(error => {
            console.error('重试场景失败:', error);
            alert('重试场景失败: ' + error.message);
        });
    }
}

// 删除场景
function deleteScene(sceneId) {
    if (confirm('确定要删除此场景吗？此操作不可撤销。')) {
        fetch('/api/scenes/' + sceneId, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert('场景已删除');
                if (currentSelectedChapterId) {
                    loadScenesList(currentSelectedChapterId); // 刷新场景列表
                }
            } else {
                alert('删除失败: ' + data.message);
            }
        })
        .catch(error => {
            console.error('删除场景失败:', error);
            alert('删除场景失败: ' + error.message);
        });
    }
}

// 重试章节工作流
function retryChapter() {
    if (!currentSelectedChapterId) {
        alert('请先选择一个章节');
        return;
    }

    if (confirm('确定要重试此章节的工作流吗？这将重新执行章节处理流程。')) {
        fetch('/api/chapters/' + currentSelectedChapterId + '/retry', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({})
        })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert('章节重试已启动');
                loadScenesList(currentSelectedChapterId); // 刷新场景列表
            } else {
                alert('重试章节失败: ' + data.message);
            }
        })
        .catch(error => {
            console.error('重试章节失败:', error);
            alert('重试章节失败: ' + error.message);
        });
    }
}

// 重新生成章节场景
function regenerateChapterScenes() {
    if (!currentSelectedChapterId) {
        alert('请先选择一个章节');
        return;
    }

    if (confirm('确定要重新生成此章节的所有场景吗？这将重新分析章节内容并生成新的场景。')) {
        fetch('/api/chapters/' + currentSelectedChapterId + '/regenerate-scenes', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({})
        })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                alert('重新生成已启动');
                loadScenesList(currentSelectedChapterId); // 刷新场景列表
            } else {
                alert('重新生成失败: ' + data.message);
            }
        })
        .catch(error => {
            console.error('重新生成章节场景失败:', error);
            alert('重新生成章节场景失败: ' + error.message);
        });
    }
}

// 刷新场景列表
function refreshScenesList() {
    if (currentSelectedChapterId) {
        loadScenesList(currentSelectedChapterId);
    } else {
        alert('请先选择一个章节');
    }
}