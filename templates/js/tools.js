// 工具相关功能函数

function loadToolsList() {
    fetch('/api/tools')
        .then(response => response.json())
        .then(tools => {
            const toolsListDiv = document.getElementById('tools-list');
            if (!toolsListDiv) return;
            
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
            
            tools.forEach(function(tool) {
                const toolCard = document.createElement('div');
                toolCard.className = 'glass-effect rounded-2xl p-6 border border-white border-opacity-20 card-hover future-glow';
                
                // 获取对应工具的图标
                const iconClass = toolIcons[tool.name] || toolIcons['default'];
                
                // 为generate_indextts2_audio工具添加特殊处理
                let cardContent = '';
                if (tool.name === 'generate_indextts2_audio') {
                    // 为音频生成工具添加表单
                    cardContent = '<div class="tool-header text-center">' +
                        '<div class="flex flex-col items-center">' +
                            '<div class="w-20 h-20 bg-blue-500 bg-opacity-20 rounded-full flex items-center justify-center mb-4">' +
                                '<i class="' + iconClass + ' text-blue-300 text-2xl"></i>' +
                            '</div>' +
                            '<h3 class="text-xl font-bold text-white mb-3">' + tool.name + '</h3>' +
                            '<p class="text-lg text-gray-300 mb-6">' + tool.description + '</p>' +
                            '<button onclick="toggleAudioForm(\'' + tool.name + '\')" class="w-full bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">' +
                                '执行工具' +
                            '</button>' +
                        '</div>' +
                    '</div>' +
                    '<div id="form_' + tool.name + '" class="audio-tool-form mt-6 p-6 glass-effect rounded-xl border border-white border-opacity-20 hidden">' +
                        '<div class="form-group mb-6">' +
                            '<label class="block text-lg font-bold text-gray-300 mb-3">输入文本:</label>' +
                            '<textarea id="textInput_' + tool.name + '" placeholder="请输入要转换为语音的文本" rows="3" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg"></textarea>' +
                        '</div>' +
                        '<div class="form-group mb-6">' +
                            '<label class="block text-lg font-bold text-gray-300 mb-3">输出目录:</label>' +
                            '<input type="text" id="outputDir_' + tool.name + '" value="./output/" placeholder="请输入输出目录" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">' +
                        '</div>' +
                        '<div class="flex flex-wrap gap-4">' +
                            '<button onclick="executeAudioTool(\'' + tool.name + '\')" class="bg-gradient-to-r from-green-500 to-green-600 hover:from-green-600 hover:to-green-700 text-white px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">' +
                                '生成音频' +
                            '</button>' +
                            '<button onclick="hideAudioForm(\'' + tool.name + '\')" class="bg-gradient-to-r from-gray-500 to-gray-600 hover:from-gray-600 hover:to-gray-700 text-white px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">' +
                                '取消' +
                            '</button>' +
                        '</div>' +
                    '</div>';
                } else if (tool.name === 'generate_images_from_chapter_with_ai_prompt') {
                    // 为图像生成工具添加表单
                    cardContent = '<div class="tool-header text-center">' +
                        '<div class="flex flex-col items-center">' +
                            '<div class="w-20 h-20 bg-blue-500 bg-opacity-20 rounded-full flex items-center justify-center mb-4">' +
                                '<i class="' + iconClass + ' text-blue-300 text-2xl"></i>' +
                            '</div>' +
                            '<h3 class="text-xl font-bold text-white mb-2 break-words max-w-full">' + tool.name + '</h3>' +
                            '<p class="text-lg text-gray-300 mb-6">' + tool.description + '</p>' +
                            '<button onclick="toggleImageForm(\'' + tool.name + '\')" class="w-full bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">' +
                                '执行工具' +
                            '</button>' +
                        '</div>' +
                    '</div>' +
                    '<div id="form_' + tool.name + '" class="audio-tool-form mt-6 p-6 glass-effect rounded-xl border border-white border-opacity-20 hidden">' +
                        '<div class="form-group mb-6">' +
                            '<label class="block text-lg font-bold text-gray-300 mb-3">章节文本:</label>' +
                            '<textarea id="chapterText_' + tool.name + '" placeholder="请输入要生成图像的章节文本" rows="4" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg"></textarea>' +
                        '</div>' +
                        '<div class="form-group mb-6">' +
                            '<label class="block text-lg font-bold text-gray-300 mb-3">输出目录:</label>' +
                            '<input type="text" id="outputDir_' + tool.name + '" value="./output/images_" placeholder="请输入输出目录" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">' +
                        '</div>' +
                        '<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">' +
                            '<div class="form-group">' +
                                '<label class="block text-lg font-bold text-gray-300 mb-3">图像宽度:</label>' +
                                '<input type="number" id="imageWidth_' + tool.name + '" value="512" min="256" max="2048" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">' +
                            '</div>' +
                            '<div class="form-group">' +
                                '<label class="block text-lg font-bold text-gray-300 mb-3">图像高度:</label>' +
                                '<input type="number" id="imageHeight_' + tool.name + '" value="896" min="256" max="2048" class="w-full px-4 py-3 bg-black bg-opacity-30 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-lg">' +
                            '</div>' +
                        '</div>' +
                        '<div class="flex flex-wrap gap-4">' +
                            '<button onclick="executeImageTool(\'' + tool.name + '\')" class="bg-gradient-to-r from-gray-200 to-gray-300 hover:from-gray-300 hover:to-gray-400 text-gray-800 px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">' +
                                '生成图像' +
                            '</button>' +
                            '<button onclick="hideImageForm(\'' + tool.name + '\')" class="bg-gradient-to-r from-gray-200 to-gray-300 hover:from-gray-300 hover:to-gray-400 text-gray-800 px-6 py-3 rounded-lg transition-all duration-200 future-glow text-lg">' +
                                '取消' +
                            '</button>' +
                        '</div>' +
                    '</div>';
                } else {
                    // 其他工具使用普通按钮
                    cardContent = '<div class="tool-header text-center">' +
                        '<div class="flex flex-col items-center">' +
                            '<div class="w-20 h-20 bg-blue-500 bg-opacity-20 rounded-full flex items-center justify-center mb-5">' +
                                '<i class="' + iconClass + ' text-blue-300 text-3xl"></i>' +
                            '</div>' +
                            '<h3 class="text-xl font-bold text-white mb-3">' + tool.name + '</h3>' +
                            '<p class="text-lg text-gray-300 mb-6">' + tool.description + '</p>' +
                            '<button onclick="executeTool(\'' + tool.name + '\')" class="w-full bg-gradient-to-r from-gray-200 to-gray-300 hover:from-gray-300 hover:to-gray-400 text-gray-800 py-3 px-4 rounded-xl transition-all duration-200 future-glow text-lg">' +
                                '执行工具' +
                            '</button>' +
                        '</div>' +
                    '</div>';
                }
                
                toolCard.innerHTML = cardContent;
                toolsListDiv.appendChild(toolCard);
            });
        })
        .catch(error => {
            console.error('Error loading tools:', error);
            // 添加错误提示
            const toolsListDiv = document.getElementById('tools-list');
            if (toolsListDiv) {
                toolsListDiv.innerHTML = '<div class="col-span-full text-red-400 p-8 text-center text-xl"><i class="fas fa-exclamation-triangle mr-3"></i>加载工具列表失败: ' + error.message + '</div>';
            }
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