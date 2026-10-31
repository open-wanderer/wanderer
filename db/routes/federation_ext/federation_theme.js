(function () {
  var s = localStorage.getItem('pbColorScheme');
  if (s === 'dark' || (!s && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.classList.add('dark');
  }
  try {
    var pb = JSON.parse(localStorage.getItem('pbSettings') || '{}');
    var accent = (pb.meta && pb.meta.accentColor) || '#1055c9';
    document.documentElement.style.setProperty('--accent', accent);
    var r = parseInt(accent.slice(1, 3), 16), g = parseInt(accent.slice(3, 5), 16), b = parseInt(accent.slice(5, 7), 16);
    document.documentElement.style.setProperty('--accent-fade', 'rgba(' + r + ',' + g + ',' + b + ',0.1)');
  } catch (_) { }
})();
