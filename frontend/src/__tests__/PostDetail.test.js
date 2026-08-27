import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

jest.mock('../services/api', () => require('../testUtils').apiMockFactory());

import PostDetail from '../pages/PostDetail';
import { postsAPI, getEditToken } from '../services/api';
import { samplePost } from '../testUtils';

const ID = '11111111-1111-1111-1111-111111111111';

const renderDetail = () =>
  render(
    <MemoryRouter initialEntries={[`/posts/${ID}`]}>
      <Routes>
        <Route path="/posts/:id" element={<PostDetail />} />
      </Routes>
    </MemoryRouter>
  );

test('renders the post it fetched', async () => {
  postsAPI.getById.mockResolvedValue({ success: true, data: samplePost() });
  getEditToken.mockReturnValue(null);
  renderDetail();
  expect(await screen.findAllByText('Blue Jansport backpack')).not.toHaveLength(0);
  expect(screen.getByText(/Found on a library table/)).toBeInTheDocument();
});

test('fetches using the id from the route', async () => {
  postsAPI.getById.mockResolvedValue({ success: true, data: samplePost() });
  getEditToken.mockReturnValue(null);
  renderDetail();
  await waitFor(() => expect(postsAPI.getById).toHaveBeenCalledWith(ID));
});

test('does not crash when the post is missing', async () => {
  postsAPI.getById.mockRejectedValue(new Error('404'));
  getEditToken.mockReturnValue(null);
  renderDetail();
  await waitFor(() => expect(postsAPI.getById).toHaveBeenCalled());
  expect(screen.queryByText('Blue Jansport backpack')).not.toBeInTheDocument();
});
