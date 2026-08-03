import { createSignal } from 'solid-js';
import { useNavigate } from '@solidjs/router';

export function LandingPage() {
  const navigate = useNavigate();
  const [keywords, setKeywords] = createSignal<string[]>([]);
  const [keywordInput, setKeywordInput] = createSignal('');

  const addKeyword = (e: Event) => {
    e.preventDefault();
    if (keywordInput().trim() !== '') {
      setKeywords([...keywords(), keywordInput().trim().toLowerCase()]);
      setKeywordInput('');
    }
  };
  
  const removeKeyword = (kw: string) => {
    setKeywords(keywords().filter(k => k !== kw));
  };

  const handleStart = () => {
    const q = keywords().join(',');
    navigate(`/chat?tags=${encodeURIComponent(q)}`);
  };

  return (
    <div class="hero-section">
      <div class="hero-content">
        <h1 class="hero-headline">
          Connect Instantly.<br/>
          <span>No Signups.</span>
        </h1>
        <p class="hero-subtitle">
          Jump into seamless, high-quality random video chats. Meet people worldwide based on your shared interests, instantly.
        </p>

        <form onSubmit={addKeyword} style="width: 100%; display: flex; flex-direction: column; align-items: center;">
          <div class="hero-input-group">
            <input 
              type="text" 
              class="hero-input"
              value={keywordInput()} 
              onInput={(e) => setKeywordInput(e.currentTarget.value)}
              placeholder="Enter an interest (e.g. music)"
            />
            <button type="button" class="hero-start-btn" onClick={keywordInput().trim() ? addKeyword : handleStart}>
              {keywordInput().trim() ? 'Add' : 'Start Chat'}
            </button>
          </div>
          
          <div class="tags-container">
            {keywords().map(kw => (
              <span class="tag" onClick={() => removeKeyword(kw)}>{kw} &times;</span>
            ))}
          </div>
        </form>
      </div>
    </div>
  );
}
