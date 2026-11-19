// La pestaña de preparación no recibe la respuesta del arranque: si el entorno
// no redirigió a tiempo, ofrece volver al ejercicio en lugar de girar sin fin.
(function () {
  var ESPERA_MS = 90000;

  var rueda = document.getElementById('rueda');
  var titulo = document.getElementById('titulo');
  var detalle = document.getElementById('detalle');
  var volver = document.getElementById('volver');
  var enlace = document.getElementById('volver-enlace');

  // Solo rutas internas: el parámetro no puede llevar a otro sitio.
  var parametro = new URLSearchParams(window.location.search).get('volver') || '';
  enlace.setAttribute('href', /^\/(?![/\\])/.test(parametro) ? parametro : '/student');

  window.setTimeout(function () {
    rueda.hidden = true;
    titulo.textContent = 'El entorno no terminó de abrirse';
    detalle.textContent =
      'Puede que el arranque haya fallado. Volvé a la página del ejercicio para ver el aviso y probar de nuevo.';
    volver.hidden = false;
  }, ESPERA_MS);
})();
