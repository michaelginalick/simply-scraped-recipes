const input   = document.getElementById('q');
const results = document.getElementById('results');
const detail  = document.getElementById('detail');
let timer;

input.addEventListener('input', () => {
  clearTimeout(timer);
  timer = setTimeout(search, 300);
});

async function search() {
  const q = input.value.trim();
  results.innerHTML = '';
  detail.hidden = true;

  if (!q) return;

  try {
    const res = await fetch(`/search?q=${encodeURIComponent(q)}`);
    if (!res.ok) return;
    const { data } = await res.json();
    renderResults(data ?? []);
  } catch (e) {
    console.error('search failed', e);
  }
}

function renderResults(recipes) {
  results.innerHTML = '';

  if (recipes.length === 0) {
    const li = document.createElement('li');
    li.className = 'empty';
    li.textContent = 'No recipes found.';
    results.appendChild(li);
    return;
  }

  for (const recipe of recipes) {
    const li = document.createElement('li');
    li.textContent = recipe.name;
    li.addEventListener('click', () => {
      results.innerHTML = '';
      showRecipe(recipe.id);
    });
    results.appendChild(li);
  }
}

async function showRecipe(id) {
  try {
    const res = await fetch(`/recipe/${id}`);
    if (!res.ok) return;
    const { data } = await res.json();
    renderDetail(data);
  } catch (e) {
    console.error('fetch recipe failed', e);
  }
}

function copyToClipboard(text) {
  if (navigator.clipboard) {
    return navigator.clipboard.writeText(text);
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.style.cssText = 'position:fixed;opacity:0;pointer-events:none';
  document.body.appendChild(ta);
  ta.focus();
  ta.select();
  try {
    document.execCommand('copy');
    return Promise.resolve();
  } catch (e) {
    return Promise.reject(e);
  } finally {
    document.body.removeChild(ta);
  }
}

// Creates an element with text-node children to avoid XSS via innerHTML.
function el(tag, ...children) {
  const node = document.createElement(tag);
  for (const child of children) {
    node.appendChild(
      typeof child === 'string' ? document.createTextNode(child) : child
    );
  }
  return node;
}

function renderDetail(recipe) {
  detail.hidden = false;
  detail.innerHTML = '';

  const header = document.createElement('div');
  header.className = 'recipe-header';
  header.appendChild(el('h2', recipe.name));

  const copyBtn = document.createElement('button');
  copyBtn.className = 'copy-btn';
  copyBtn.textContent = 'Copy link';
  copyBtn.addEventListener('click', () => {
    const url = `${window.location.origin}/r/${recipe.id}`;
    copyToClipboard(url).then(() => {
      copyBtn.textContent = 'Copied!';
      copyBtn.classList.add('copied');
      setTimeout(() => {
        copyBtn.textContent = 'Copy link';
        copyBtn.classList.remove('copied');
      }, 2000);
    });
  });
  header.appendChild(copyBtn);
  detail.appendChild(header);

  detail.appendChild(el('h3', 'Ingredients'));
  const ul = document.createElement('ul');
  for (const ing of recipe.ingredients ?? []) {
    ul.appendChild(el('li', ing.name));
  }
  detail.appendChild(ul);

  detail.appendChild(el('h3', 'Instructions'));
  const ol = document.createElement('ol');
  for (const inst of recipe.instructions ?? []) {
    ol.appendChild(el('li', inst.step));
  }
  detail.appendChild(ol);
}
