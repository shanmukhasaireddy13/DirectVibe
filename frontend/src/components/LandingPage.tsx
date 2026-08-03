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
    <div class="landing-page">
      <div class="landing-card">
        <div class="hero-text">
          <h1>DirectVibe</h1>
          <p>Instant, random video chat.</p>
        </div>
        
        <form onSubmit={addKeyword} style="display: flex; flex-direction: column; gap: 1rem;">
          <div class="input-group">
            <input 
              type="text" 
              value={keywordInput()} 
              onInput={(e) => setKeywordInput(e.currentTarget.value)}
              placeholder="Interests (e.g. gaming, music)"
            />
            <button type="submit" class="btn">Add</button>
          </div>
          
          <div class="tags-container">
            {keywords().map(kw => (
              <span class="tag" onClick={() => removeKeyword(kw)}>{kw} &times;</span>
            ))}
          </div>

          <button 
            type="button" 
            class="btn" 
            style="width: 100%; font-size: 1.125rem; padding: 1rem;" 
            onClick={handleStart}
          >
            Start Chat
          </button>
        </form>
      </div>
    </div>
  );
}
