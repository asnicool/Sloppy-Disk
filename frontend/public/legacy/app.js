// Legacy Music Player - ES5 Compatible JavaScript
// Works on iOS 10.3 (Safari 10)

(function() {
  'use strict';

  var API_BASE = '';
  var statusInterval = null;
  var currentStatus = null;

  // DOM Elements
  var statusEl, currentSongEl, elapsedEl;
  var btnPlay, btnPause, btnPrev, btnNext;
  var searchInput, searchResults, albumsGrid, queueList;

  // Initialize
  function init() {
    statusEl = document.getElementById('play-status');
    currentSongEl = document.getElementById('current-song');
    elapsedEl = document.getElementById('elapsed-time');
    btnPlay = document.getElementById('btn-play');
    btnPause = document.getElementById('btn-pause');
    btnPrev = document.getElementById('btn-prev');
    btnNext = document.getElementById('btn-next');
    searchInput = document.getElementById('search-input');
    searchResults = document.getElementById('search-results');
    albumsGrid = document.getElementById('albums-grid');
    queueList = document.getElementById('queue-list');

    bindEvents();
    loadRandomAlbums();
    loadQueue();
    startStatusPolling();
  }

  function bindEvents() {
    btnPlay.onclick = handlePlay;
    btnPause.onclick = handlePause;
    btnPrev.onclick = handlePrev;
    btnNext.onclick = handleNext;
    document.getElementById('btn-search').onclick = handleSearch;
    searchInput.onkeypress = function(e) {
      if (e.key === 'Enter') {
        handleSearch();
      }
    };
  }

  // API Helpers (XMLHttpRequest for iOS 10.3)
  function apiGet(url, callback) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', API_BASE + url, true);
    xhr.onreadystatechange = function() {
      if (xhr.readyState === 4) {
        if (xhr.status === 200) {
          try {
            var data = JSON.parse(xhr.responseText);
            callback(null, data);
          } catch (e) {
            callback(e, null);
          }
        } else {
          callback(new Error('HTTP ' + xhr.status), null);
        }
      }
    };
    xhr.send();
  }

  function apiPost(url, callback) {
    var xhr = new XMLHttpRequest();
    xhr.open('POST', API_BASE + url, true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onreadystatechange = function() {
      if (xhr.readyState === 4) {
        if (xhr.status === 200 || xhr.status === 204) {
          callback(null, null);
        } else {
          callback(new Error('HTTP ' + xhr.status), null);
        }
      }
    };
    xhr.send(JSON.stringify({}));
  }

  // Status Polling
  function startStatusPolling() {
    updateStatus();
    statusInterval = setInterval(updateStatus, 3000);
  }

  function updateStatus() {
    apiGet('/api/status', function(err, data) {
      if (err) {
        updateStatusDisplay('error', '', '');
        return;
      }
      currentStatus = data;
      var state = data.state || 'stop';
      var song = data.currentSong || '';
      var elapsed = formatTime(data.elapsed);
      updateStatusDisplay(state, song, elapsed);
      updateControls(state);
    });
  }

  function updateStatusDisplay(state, song, elapsed) {
    if (!statusEl) return;
    if (state === 'play') {
      statusEl.textContent = 'Playing';
      statusEl.className = 'playing';
    } else if (state === 'pause') {
      statusEl.textContent = 'Paused';
      statusEl.className = 'paused';
    } else {
      statusEl.textContent = 'Stopped';
      statusEl.className = 'stopped';
    }
    if (currentSongEl) {
      currentSongEl.textContent = song;
    }
    if (elapsedEl) {
      elapsedEl.textContent = elapsed;
    }
  }

  function updateControls(state) {
    if (!btnPlay || !btnPause) return;
    if (state === 'play') {
      btnPlay.style.display = 'none';
      btnPause.style.display = 'inline-block';
    } else {
      btnPlay.style.display = 'inline-block';
      btnPause.style.display = 'none';
    }
  }

  function formatTime(seconds) {
    if (!seconds) return '0:00';
    var mins = Math.floor(seconds / 60);
    var secs = Math.floor(seconds % 60);
    return mins + ':' + (secs < 10 ? '0' : '') + secs;
  }

  // Control Handlers
  function handlePlay() {
    apiPost('/api/play', function(err) {
      if (!err) updateStatus();
    });
  }

  function handlePause() {
    apiPost('/api/pause', function(err) {
      if (!err) updateStatus();
    });
  }

  function handlePrev() {
    apiPost('/api/previous', function(err) {
      if (!err) updateStatus();
    });
  }

  function handleNext() {
    apiPost('/api/next', function(err) {
      if (!err) updateStatus();
    });
  }

  // Search
  function handleSearch() {
    var query = searchInput.value.trim();
    if (!query) return;
    searchResults.innerHTML = '<p class="loading">Searching...</p>';
    apiGet('/api/search?q=' + encodeURIComponent(query), function(err, data) {
      if (err) {
        searchResults.innerHTML = '<p class="empty">Error: ' + err.message + '</p>';
        return;
      }
      renderSearchResults(data || []);
    });
  }

  function renderSearchResults(results) {
    if (!results || results.length === 0) {
      searchResults.innerHTML = '<p class="empty">No results found</p>';
      return;
    }
    var html = '';
    var i, item;
    for (i = 0; i < results.length; i++) {
      item = results[i];
      html += renderResultItem(item);
    }
    searchResults.innerHTML = html;
  }

  function renderResultItem(item) {
    var title = item.title || item.Album || 'Unknown';
    var artist = item.artist || 'Unknown Artist';
    var path = item.path || '';
    return '<div class="result-item">' +
      '<div class="result-info">' +
        '<div class="result-title">' + escapeHtml(title) + '</div>' +
        '<div class="result-artist">' + escapeHtml(artist) + '</div>' +
      '</div>' +
      '<div class="result-actions">' +
        '<button class="btn btn-small" onclick="legacyApp.enqueue(\'' + escapeChars(path) + '\')">Enqueue</button>' +
      '</div>' +
    '</div>';
  }

  // Random Albums
  function loadRandomAlbums() {
    apiGet('/api/albums/random', function(err, data) {
      if (err) {
        albumsGrid.innerHTML = '<p class="empty">Error loading albums</p>';
        return;
      }
      renderAlbums(data || []);
    });
  }

  function renderAlbums(albums) {
    if (!albums || albums.length === 0) {
      albumsGrid.innerHTML = '<p class="empty">No albums found</p>';
      return;
    }
    var html = '';
    var i, album;
    for (i = 0; i < albums.length; i++) {
      album = albums[i];
      html += renderAlbumCard(album);
    }
    albumsGrid.innerHTML = html;
  }

  function renderAlbumCard(album) {
    var title = album.album || album.Album || 'Unknown Album';
    var artist = album.artist || 'Unknown Artist';
    var coverUrl = album.coverart || '';
    var albumKey = album.album || '';
    var artistKey = album.artist || '';

    var coverHtml = coverUrl ? '<img src="' + escapeHtml(coverUrl) + '" alt="">' : 'No Cover';

    return '<div class="album-card">' +
      '<div class="album-cover">' + coverHtml + '</div>' +
      '<div class="album-title" title="' + escapeHtml(title) + '">' + escapeHtml(title) + '</div>' +
      '<div class="album-artist" title="' + escapeHtml(artist) + '">' + escapeHtml(artist) + '</div>' +
      '<div class="album-actions">' +
        '<button class="btn btn-small" onclick="legacyApp.playAlbum(\'' + escapeChars(artistKey) + '\',\'' + escapeChars(albumKey) + '\')">Play</button>' +
        '<button class="btn btn-small btn-queue" onclick="legacyApp.enqueueAlbum(\'' + escapeChars(artistKey) + '\',\'' + escapeChars(albumKey) + '\')">+</button>' +
      '</div>' +
    '</div>';
  }

  // Queue
  function loadQueue() {
    apiGet('/api/playlist', function(err, data) {
      if (err) {
        queueList.innerHTML = '<p class="empty">Error loading queue</p>';
        return;
      }
      renderQueue(data || []);
    });
  }

  function renderQueue(items) {
    if (!items || items.length === 0) {
      queueList.innerHTML = '<p class="empty">Queue is empty</p>';
      return;
    }
    var html = '';
    var i, item;
    var pos = currentStatus && currentStatus.playlistPos ? currentStatus.playlistPos : -1;
    for (i = 0; i < items.length; i++) {
      item = items[i];
      html += renderQueueItem(item, i, i === pos);
    }
    queueList.innerHTML = html;
  }

  function renderQueueItem(item, index, isActive) {
    var title = item.title || 'Unknown';
    var artist = item.artist || 'Unknown Artist';
    var activeClass = isActive ? ' active' : '';
    return '<div class="queue-item' + activeClass + '">' +
      '<div class="queue-pos">' + (index + 1) + '</div>' +
      '<div class="queue-info">' +
        '<div class="queue-title">' + escapeHtml(title) + '</div>' +
        '<div class="queue-artist">' + escapeHtml(artist) + '</div>' +
      '</div>' +
    '</div>';
  }

  // Actions
  function enqueue(path) {
    var url = '/api/add?path=' + encodeURIComponent(path);
    apiGet(url, function(err) {
      if (!err) {
        loadQueue();
        showMessage('Added to queue');
      }
    });
  }

  function enqueueAlbum(artist, album) {
    var url = '/api/search?artist=' + encodeURIComponent(artist) + '&album=' + encodeURIComponent(album);
    apiGet(url, function(err, data) {
      if (!err && data && data.length > 0) {
        var i, item;
        for (i = 0; i < data.length; i++) {
          enqueue(data[i].path);
        }
        showMessage('Album added to queue');
      }
    });
  }

  function playAlbum(artist, album) {
    enqueueAlbum(artist, album);
    setTimeout(function() {
      apiPost('/api/play', function() {
        updateStatus();
      });
    }, 500);
  }

  function showMessage(msg) {
    var el = document.createElement('div');
    el.className = 'message';
    el.textContent = msg;
    el.style.cssText = 'position:fixed;bottom:20px;left:50%;transform:translateX(-50%);background:#333;color:#fff;padding:10px 20px;border-radius:5px;z-index:9999;';
    document.body.appendChild(el);
    setTimeout(function() {
      document.body.removeChild(el);
    }, 2000);
  }

  // Utility Functions
  function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function escapeChars(str) {
    if (!str) return '';
    return str.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
  }

  // Expose global API
  window.legacyApp = {
    enqueue: enqueue,
    enqueueAlbum: enqueueAlbum,
    playAlbum: playAlbum,
    loadQueue: loadQueue,
    updateStatus: updateStatus
  };

  // Start when DOM ready
  if (document.readyState === 'loading') {
    document.onreadystatechange = function() {
      if (document.readyState === 'complete') {
        init();
      }
    };
  } else {
    init();
  }

})();