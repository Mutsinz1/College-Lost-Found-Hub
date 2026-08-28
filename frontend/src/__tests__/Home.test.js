import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

jest.mock('../services/api', () => require('../testUtils').apiMockFactory());

import Home from '../pages/Home';
import { postsAPI, buildingsAPI, areasAPI } from '../services/api';
import { samplePost } from '../testUtils';

function wireAPI({ posts = [samplePost()], searchFails = false } = {}) {
  if (searchFails) {
    postsAPI.search.mockRejectedValue(new Error('backend down'));
  } else {
    postsAPI.search.mockResolvedValue({ success: true, data: { posts, total: posts.length } });
  }
  buildingsAPI.getAll.mockResolvedValue({ success: true, data: { buildings: [] } });
  areasAPI.getAll.mockResolvedValue({ success: true, data: { areas: [] } });
}

const renderHome = () => render(<MemoryRouter><Home /></MemoryRouter>);

test('renders a post returned by the API', async () => {
  wireAPI();
  renderHome();
  expect(await screen.findAllByText('Blue Jansport backpack')).not.toHaveLength(0);
});

test('renders the map', async () => {
  wireAPI();
  renderHome();
  await waitFor(() => expect(screen.getByTestId('map')).toBeInTheDocument());
});

test('searches using the geolocation position', async () => {
  wireAPI();
  renderHome();
  await waitFor(() => expect(postsAPI.search).toHaveBeenCalled());
  const params = postsAPI.search.mock.calls[0][0];
  expect(params.lat).toBeCloseTo(40.7128);
  expect(params.lng).toBeCloseTo(-74.006);
});

test('shows an empty map rather than crashing when the API fails', async () => {
  wireAPI({ searchFails: true });
  renderHome();
  await waitFor(() => expect(postsAPI.search).toHaveBeenCalled());
  expect(screen.getByTestId('map')).toBeInTheDocument();
});

test('tolerates a malformed response without crashing', async () => {
  postsAPI.search.mockResolvedValue({ success: true, data: null });
  buildingsAPI.getAll.mockResolvedValue({ success: true, data: { buildings: [] } });
  areasAPI.getAll.mockResolvedValue({ success: true, data: { areas: [] } });
  renderHome();
  await waitFor(() => expect(postsAPI.search).toHaveBeenCalled());
  expect(screen.getByTestId('map')).toBeInTheDocument();
});

test('renders on an empty database, where the API returns null collections', async () => {
  // A nil Go slice marshals to JSON null. Before this was guarded, .map() on
  // null threw during render, unmounted the tree, and every fresh deployment
  // served a blank page.
  postsAPI.search.mockResolvedValue({ success: true, data: { posts: null, total: 0 } });
  buildingsAPI.getAll.mockResolvedValue({ success: true, data: { buildings: null } });
  areasAPI.getAll.mockResolvedValue({ success: true, data: { areas: null } });

  renderHome();
  await waitFor(() => expect(postsAPI.search).toHaveBeenCalled());
  expect(screen.getByTestId('map')).toBeInTheDocument();
  expect(screen.getByText(/Campus Map/)).toBeInTheDocument();
});
