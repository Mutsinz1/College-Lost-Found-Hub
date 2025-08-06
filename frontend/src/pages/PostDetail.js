import React from 'react';
import { useParams } from 'react-router-dom';

const PostDetail = () => {
  const { id } = useParams();

  return (
    <div className="max-w-4xl mx-auto">
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <h1 className="text-2xl font-bold text-gray-900 mb-4">Post Details</h1>
        <p className="text-gray-600 mb-4">
          Post ID: {id}
        </p>
        <p className="text-gray-600">
          This page will show detailed information about a specific post, including images, 
          contact information, and interaction options.
        </p>
        <div className="mt-4 p-4 bg-blue-50 rounded-md">
          <p className="text-blue-800">
            <strong>Coming Soon:</strong> Detailed post view with images, contact forms, and interaction buttons.
          </p>
        </div>
      </div>
    </div>
  );
};

export default PostDetail; 