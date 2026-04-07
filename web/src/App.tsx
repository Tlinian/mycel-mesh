import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Login from './pages/Login';
import Nodes from './pages/Nodes';
import Layout from './components/Layout';

const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={<Layout />}>
          <Route index element={<Nodes />} />
          <Route path="nodes" element={<Nodes />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
};

export default App;
