import { Router, Route } from '@solidjs/router';
import { LandingPage } from './components/LandingPage';
import { ChatRoom } from './components/ChatRoom';

function App() {
  return (
    <Router root={(props) => <div class="app-container">{props.children}</div>}>
      <Route path="/" component={LandingPage} />
      <Route path="/chat" component={ChatRoom} />
    </Router>
  );
}

export default App;
