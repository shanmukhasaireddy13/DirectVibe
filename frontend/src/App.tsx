import { Router, Route } from '@solidjs/router';
import { LandingPage } from './components/LandingPage';
import { ChatRoom } from './components/ChatRoom';
import { Header } from './components/Header';
import { Footer } from './components/Footer';
import { PrivacyPolicy } from './components/PrivacyPolicy';
import { TermsOfUse } from './components/TermsOfUse';
import { onMount } from 'solid-js';
import { initTheme } from './lib/theme';

function App() {
  onMount(() => {
    initTheme();
  });

  return (
    <Router root={(props) => (
      <div class="app-container">
        <Header />
        <main class="main-content-wrapper">
          {props.children}
        </main>
        <Footer />
      </div>
    )}>
      <Route path="/" component={LandingPage} />
      <Route path="/chat" component={ChatRoom} />
      <Route path="/privacy" component={PrivacyPolicy} />
      <Route path="/terms" component={TermsOfUse} />
    </Router>
  );
}

export default App;
