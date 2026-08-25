import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import {
  ArrowLeft, MapPin, Calendar, Clock, User, Building, Mail,
  Pencil, Trash2, HandHelping, CheckCircle, XCircle, Inbox,
} from 'lucide-react';
import toast from 'react-hot-toast';
import { postsAPI, interactionsAPI, getEditToken, removeEditToken } from '../services/api';

const categoryIcons = { pet: '🐕', document: '📄', item: '📱', other: '❓' };

const statusStyles = {
  active: 'bg-success-100 text-success-800',
  claimed: 'bg-yellow-100 text-yellow-800',
  resolved: 'bg-gray-100 text-gray-800',
};

const PostDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();

  const [post, setPost] = useState(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [selectedImage, setSelectedImage] = useState(0);

  // Owner state (present when this browser created the post)
  const editToken = getEditToken(id);
  const isOwner = Boolean(editToken);
  const [interactions, setInteractions] = useState([]);
  const [showEdit, setShowEdit] = useState(false);
  const [editForm, setEditForm] = useState({ title: '', description: '', status: '' });

  // Claim form state
  const [showClaimForm, setShowClaimForm] = useState(false);
  const [claimForm, setClaimForm] = useState({ contact_name: '', contact_email: '', message: '' });
  const [submittingClaim, setSubmittingClaim] = useState(false);

  const loadPost = useCallback(async () => {
    try {
      const response = await postsAPI.getById(id);
      if (response.success) {
        setPost(response.data);
        setEditForm({
          title: response.data.title || '',
          description: response.data.description || '',
          status: response.data.status || 'active',
        });
      } else {
        setNotFound(true);
      }
    } catch (error) {
      if (error.response && error.response.status === 404) {
        setNotFound(true);
      } else {
        toast.error('Failed to load post');
      }
    } finally {
      setLoading(false);
    }
  }, [id]);

  const loadInteractions = useCallback(async () => {
    if (!getEditToken(id)) return;
    try {
      const response = await interactionsAPI.listForPost(id);
      if (response.success) {
        setInteractions(response.data.interactions || []);
      }
    } catch (error) {
      console.error('Failed to load interactions:', error);
    }
  }, [id]);

  useEffect(() => {
    loadPost();
    loadInteractions();
  }, [loadPost, loadInteractions]);

  const handleClaimSubmit = async (e) => {
    e.preventDefault();
    if (!claimForm.contact_email.includes('@')) {
      toast.error('Please enter a valid email address');
      return;
    }
    setSubmittingClaim(true);
    try {
      const response = await interactionsAPI.create(id, {
        interaction_type: 'claim',
        ...claimForm,
      });
      if (response.success) {
        toast.success('Your claim was sent to the poster. They will contact you by email.');
        setShowClaimForm(false);
        setClaimForm({ contact_name: '', contact_email: '', message: '' });
      } else {
        toast.error(response.error || 'Failed to submit claim');
      }
    } catch (error) {
      const msg = error.response?.data?.error || 'Failed to submit claim';
      toast.error(msg);
    } finally {
      setSubmittingClaim(false);
    }
  };

  const handleEditSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await postsAPI.update(id, editForm);
      if (response.success) {
        toast.success('Post updated');
        setShowEdit(false);
        loadPost();
      } else {
        toast.error(response.error || 'Failed to update post');
      }
    } catch (error) {
      toast.error(error.response?.data?.error || 'Failed to update post');
    }
  };

  const handleDelete = async () => {
    // eslint-disable-next-line no-restricted-globals
    if (!window.confirm('Delete this post? This cannot be undone.')) return;
    try {
      const response = await postsAPI.delete(id);
      if (response.success) {
        removeEditToken(id);
        toast.success('Post deleted');
        navigate('/');
      } else {
        toast.error(response.error || 'Failed to delete post');
      }
    } catch (error) {
      toast.error(error.response?.data?.error || 'Failed to delete post');
    }
  };

  const handleInteractionDecision = async (interactionId, status) => {
    try {
      const response = await interactionsAPI.updateStatus(interactionId, status, id);
      if (response.success) {
        toast.success(status === 'accepted' ? 'Claim accepted — the item is now marked as claimed.' : 'Claim rejected.');
        loadInteractions();
        loadPost();
      } else {
        toast.error(response.error || 'Failed to update claim');
      }
    } catch (error) {
      toast.error(error.response?.data?.error || 'Failed to update claim');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="spinner"></div>
        <span className="ml-3 text-gray-600">Loading post...</span>
      </div>
    );
  }

  if (notFound || !post) {
    return (
      <div className="max-w-2xl mx-auto text-center py-16">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Post not found</h1>
        <p className="text-gray-600 mb-6">This post may have expired or been removed.</p>
        <Link to="/" className="inline-flex items-center space-x-2 px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors">
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Home</span>
        </Link>
      </div>
    );
  }

  const images = post.image_urls || [];
  const area = post.lost_found_area;

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Back link */}
      <Link to="/" className="inline-flex items-center space-x-2 text-sm text-gray-600 hover:text-gray-900 transition-colors">
        <ArrowLeft className="w-4 h-4" />
        <span>Back to all items</span>
      </Link>

      {/* Main card */}
      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        {/* Image gallery */}
        {images.length > 0 && (
          <div>
            <img
              src={images[selectedImage]}
              alt={post.title}
              className="w-full max-h-96 object-contain bg-gray-100"
            />
            {images.length > 1 && (
              <div className="flex space-x-2 p-3 overflow-x-auto">
                {images.map((url, i) => (
                  <button
                    key={url}
                    onClick={() => setSelectedImage(i)}
                    className={`w-16 h-16 rounded-md overflow-hidden border-2 flex-shrink-0 ${
                      i === selectedImage ? 'border-primary-600' : 'border-transparent'
                    }`}
                  >
                    <img src={url} alt={`${post.title} ${i + 1}`} className="w-full h-full object-cover" />
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        <div className="p-6">
          {/* Title + badges */}
          <div className="flex flex-wrap items-center gap-2 mb-2">
            <span className="text-3xl">{categoryIcons[post.category] || '❓'}</span>
            <h1 className="text-2xl font-bold text-gray-900">{post.title}</h1>
          </div>
          <div className="flex flex-wrap items-center gap-2 mb-4">
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              post.is_lost_item ? 'bg-danger-100 text-danger-800' : 'bg-success-100 text-success-800'
            }`}>
              {post.is_lost_item ? 'Lost Item' : 'Found Item'}
            </span>
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${statusStyles[post.status] || 'bg-gray-100 text-gray-800'}`}>
              {post.status.charAt(0).toUpperCase() + post.status.slice(1)}
            </span>
            <span className="px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-700 capitalize">
              {post.category}
            </span>
          </div>

          {post.description && (
            <p className="text-gray-700 whitespace-pre-wrap mb-6">{post.description}</p>
          )}

          {/* Meta */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm text-gray-600 mb-6">
            <div className="flex items-center space-x-2">
              <Calendar className="w-4 h-4 text-gray-400" />
              <span>Posted {new Date(post.created_at).toLocaleDateString()}</span>
            </div>
            <div className="flex items-center space-x-2">
              <Clock className="w-4 h-4 text-gray-400" />
              <span>Expires {new Date(post.expires_at).toLocaleDateString()}</span>
            </div>
            {post.poster_name && (
              <div className="flex items-center space-x-2">
                <User className="w-4 h-4 text-gray-400" />
                <span>Posted by {post.poster_name}</span>
              </div>
            )}
            {post.contact_email && (
              <div className="flex items-center space-x-2">
                <Mail className="w-4 h-4 text-gray-400" />
                <a href={`mailto:${post.contact_email}`} className="text-primary-600 hover:underline">
                  {post.contact_email}
                </a>
              </div>
            )}
          </div>

          {/* Pickup info */}
          {area && (
            <div className="bg-blue-50 border border-blue-200 rounded-md p-4 mb-6">
              <div className="flex items-center space-x-2 mb-2">
                <Building className="w-5 h-5 text-blue-700" />
                <h2 className="font-semibold text-blue-900">
                  Pickup: {area.name}{area.building ? ` — ${area.building.name}` : ''}
                </h2>
              </div>
              <div className="text-sm text-blue-800 space-y-1">
                {area.location_description && <p>📍 {area.location_description}</p>}
                {area.contact_person && <p>👤 Contact: {area.contact_person}</p>}
                {area.hours_of_operation && <p>🕐 Hours: {area.hours_of_operation}</p>}
                {area.pickup_instructions && <p>ℹ️ {area.pickup_instructions}</p>}
              </div>
            </div>
          )}

          {/* Claim call-to-action */}
          {post.status === 'active' && !isOwner && (
            <div className="mb-2">
              {!showClaimForm ? (
                <button
                  onClick={() => setShowClaimForm(true)}
                  className="inline-flex items-center space-x-2 px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors"
                >
                  <HandHelping className="w-4 h-4" />
                  <span>{post.is_lost_item ? "I found this item" : 'This is mine'}</span>
                </button>
              ) : (
                <form onSubmit={handleClaimSubmit} className="bg-gray-50 border rounded-md p-4 space-y-3">
                  <h3 className="font-semibold text-gray-900">
                    {post.is_lost_item ? 'Tell the owner you found it' : 'Claim this item'}
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <input
                      type="text"
                      placeholder="Your name"
                      value={claimForm.contact_name}
                      onChange={(e) => setClaimForm({ ...claimForm, contact_name: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                      required
                    />
                    <input
                      type="email"
                      placeholder="Your email"
                      value={claimForm.contact_email}
                      onChange={(e) => setClaimForm({ ...claimForm, contact_email: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                      required
                    />
                  </div>
                  <textarea
                    placeholder={post.is_lost_item
                      ? 'Where and when did you find it?'
                      : 'Describe the item to prove it is yours (color, contents, marks...)'}
                    value={claimForm.message}
                    onChange={(e) => setClaimForm({ ...claimForm, message: e.target.value })}
                    rows={3}
                    maxLength={2000}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                  <div className="flex space-x-2">
                    <button
                      type="submit"
                      disabled={submittingClaim}
                      className="px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors disabled:opacity-50"
                    >
                      {submittingClaim ? 'Sending...' : 'Send'}
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowClaimForm(false)}
                      className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-100 transition-colors"
                    >
                      Cancel
                    </button>
                  </div>
                </form>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Map */}
      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div className="p-4 border-b flex items-center space-x-2">
          <MapPin className="w-5 h-5 text-gray-500" />
          <h2 className="text-lg font-semibold text-gray-900">Location</h2>
        </div>
        <MapContainer
          center={[post.location.latitude, post.location.longitude]}
          zoom={17}
          style={{ height: '300px', width: '100%' }}
        >
          <TileLayer
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          />
          <Marker position={[post.location.latitude, post.location.longitude]}>
            <Popup>{post.title}</Popup>
          </Marker>
        </MapContainer>
      </div>

      {/* Owner panel */}
      {isOwner && (
        <div className="bg-white rounded-lg shadow-sm border">
          <div className="p-4 border-b flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Manage your post</h2>
            <div className="flex space-x-2">
              <button
                onClick={() => setShowEdit(!showEdit)}
                className="inline-flex items-center space-x-1 px-3 py-1.5 border border-gray-300 text-gray-700 text-sm rounded-md hover:bg-gray-100 transition-colors"
              >
                <Pencil className="w-4 h-4" />
                <span>Edit</span>
              </button>
              <button
                onClick={handleDelete}
                className="inline-flex items-center space-x-1 px-3 py-1.5 bg-danger-100 text-danger-800 text-sm rounded-md hover:bg-danger-200 transition-colors"
              >
                <Trash2 className="w-4 h-4" />
                <span>Delete</span>
              </button>
            </div>
          </div>

          {showEdit && (
            <form onSubmit={handleEditSubmit} className="p-4 border-b space-y-3">
              <input
                type="text"
                value={editForm.title}
                onChange={(e) => setEditForm({ ...editForm, title: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="Title"
              />
              <textarea
                value={editForm.description}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                rows={3}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="Description"
              />
              <select
                value={editForm.status}
                onChange={(e) => setEditForm({ ...editForm, status: e.target.value })}
                className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="active">Active</option>
                <option value="claimed">Claimed</option>
                <option value="resolved">Resolved</option>
              </select>
              <div>
                <button type="submit" className="px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors">
                  Save changes
                </button>
              </div>
            </form>
          )}

          {/* Claims inbox */}
          <div className="p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Inbox className="w-5 h-5 text-gray-500" />
              <h3 className="font-semibold text-gray-900">Claims & messages ({interactions.length})</h3>
            </div>
            {interactions.length === 0 ? (
              <p className="text-sm text-gray-500">No one has responded to this post yet.</p>
            ) : (
              <div className="space-y-3">
                {interactions.map((interaction) => (
                  <div key={interaction.id} className="border rounded-md p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2 mb-1">
                      <div className="flex items-center space-x-2 text-sm">
                        <span className="font-medium text-gray-900">
                          {interaction.contact_name || 'Anonymous'}
                        </span>
                        <a href={`mailto:${interaction.contact_email}`} className="text-primary-600 hover:underline">
                          {interaction.contact_email}
                        </a>
                        <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-700 capitalize">
                          {interaction.interaction_type}
                        </span>
                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium capitalize ${
                          interaction.status === 'accepted' ? 'bg-success-100 text-success-800'
                            : interaction.status === 'rejected' ? 'bg-danger-100 text-danger-800'
                            : 'bg-yellow-100 text-yellow-800'
                        }`}>
                          {interaction.status}
                        </span>
                      </div>
                      <span className="text-xs text-gray-500">
                        {new Date(interaction.created_at).toLocaleString()}
                      </span>
                    </div>
                    {interaction.message && (
                      <p className="text-sm text-gray-700 mb-2 whitespace-pre-wrap">{interaction.message}</p>
                    )}
                    {interaction.status === 'pending' && (
                      <div className="flex space-x-2">
                        <button
                          onClick={() => handleInteractionDecision(interaction.id, 'accepted')}
                          className="inline-flex items-center space-x-1 px-3 py-1 bg-success-100 text-success-800 text-xs rounded hover:bg-success-200 transition-colors"
                        >
                          <CheckCircle className="w-3 h-3" />
                          <span>Accept</span>
                        </button>
                        <button
                          onClick={() => handleInteractionDecision(interaction.id, 'rejected')}
                          className="inline-flex items-center space-x-1 px-3 py-1 bg-danger-100 text-danger-800 text-xs rounded hover:bg-danger-200 transition-colors"
                        >
                          <XCircle className="w-3 h-3" />
                          <span>Reject</span>
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default PostDetail;
