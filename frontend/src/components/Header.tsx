import { useNavigate } from '@solidjs/router';
import { theme, toggleTheme } from '../lib/theme';

export function Header() {
  const navigate = useNavigate();
  return (
    <header class="app-header">
      <h1 onClick={() => navigate('/')}>DirectVibe</h1>
      <button class="theme-toggle-btn" onClick={toggleTheme} title="Toggle Theme">
        {theme() === 'dark' ? '☀️' : '🌙'}
      </button>
    </header>
  );
}

