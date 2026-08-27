import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

jest.mock('../services/api', () => require('../testUtils').apiMockFactory());

import Header from '../components/Header';
import { getCurrentUser } from '../services/api';

test('renders the app name and primary navigation', () => {
  getCurrentUser.mockReturnValue(null);
  render(<MemoryRouter><Header /></MemoryRouter>);
  expect(screen.getByText(/Lost & Found/i)).toBeInTheDocument();
});

test('renders without a signed-in user', () => {
  getCurrentUser.mockReturnValue(null);
  render(<MemoryRouter><Header /></MemoryRouter>);
  expect(screen.queryByText(/Sign out/i)).not.toBeInTheDocument();
});

test('shows the signed-in user when there is one', () => {
  getCurrentUser.mockReturnValue({ id: 'u1', name: 'Abel', email: 'abel@college.edu', role: 'user' });
  render(<MemoryRouter><Header /></MemoryRouter>);
  expect(screen.getByText(/Abel/)).toBeInTheDocument();
});
