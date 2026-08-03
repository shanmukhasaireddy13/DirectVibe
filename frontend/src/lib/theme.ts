import { createSignal } from 'solid-js';

const [theme, setTheme] = createSignal<'light' | 'dark'>('dark');

export { theme, setTheme };

export const initTheme = () => {
  const saved = localStorage.getItem('theme');
  if (saved === 'light') {
    setTheme('light');
    document.documentElement.classList.remove('dark');
  } else {
    setTheme('dark');
    document.documentElement.classList.add('dark');
  }
};

export const toggleTheme = () => {
  const newTheme = theme() === 'light' ? 'dark' : 'light';
  setTheme(newTheme);
  localStorage.setItem('theme', newTheme);
  if (newTheme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
};
