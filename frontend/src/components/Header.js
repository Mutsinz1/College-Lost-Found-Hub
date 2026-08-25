import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Search, Plus, Home, LogOut, User } from 'lucide-react';
import toast from 'react-hot-toast';
import { getCurrentUser, loginWithGoogle, loginDev, logout } from '../services/api';

const GOOGLE_CLIENT_ID = process.env.REACT_APP_GOOGLE_CLIENT_ID || '';

const Header = () => {
  const location = useLocation();
  const [user, setUser] = useState(getCurrentUser());
  const googleButtonRef = useRef(null);

  const isActive = (path) => {
    return location.pathname === path;
  };

  const handleGoogleCredential = useCallback(async (response) => {
    try {
      const signedInUser = await loginWithGoogle(response.credential);
      setUser(signedInUser);
      toast.success(`Welcome, ${signedInUser.name}!`);
    } catch (error) {
      toast.error(error.response?.data?.error || error.message || 'Sign-in failed');
    }
  }, []);

  // Load Google Identity Services and render the sign-in button
  useEffect(() => {
    if (!GOOGLE_CLIENT_ID || user) return;

    const renderButton = () => {
      if (window.google && googleButtonRef.current) {
        window.google.accounts.id.initialize({
          client_id: GOOGLE_CLIENT_ID,
          callback: handleGoogleCredential,
        });
        window.google.accounts.id.renderButton(googleButtonRef.current, {
          theme: 'outline',
          size: 'medium',
          text: 'signin_with',
        });
      }
    };

    if (window.google) {
      renderButton();
      return;
    }

    const script = document.createElement('script');
    script.src = 'https://accounts.google.com/gsi/client';
    script.async = true;
    script.onload = renderButton;
    document.head.appendChild(script);
  }, [user, handleGoogleCredential]);

  const handleDevLogin = async () => {
    const email = window.prompt('Enter your school email to sign in (dev mode):');
    if (!email) return;
    try {
      const signedInUser = await loginDev(email.trim());
      setUser(signedInUser);
      toast.success(`Welcome, ${signedInUser.name}!`);
    } catch (error) {
      toast.error(error.response?.data?.error || error.message || 'Sign-in failed');
    }
  };

  const handleLogout = () => {
    logout();
    setUser(null);
    toast.success('Signed out');
  };

  return (
    <header className="bg-white shadow-sm border-b">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/" className="flex items-center space-x-2">
            <div className="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
              <Search className="w-5 h-5 text-white" />
            </div>
            <span className="text-xl font-bold text-gray-900">Lost & Found</span>
          </Link>

          {/* Navigation */}
          <nav className="flex items-center space-x-4">
            <Link
              to="/"
              className={`flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                isActive('/')
                  ? 'bg-primary-100 text-primary-700'
                  : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
              }`}
            >
              <Home className="w-4 h-4" />
              <span>Home</span>
            </Link>

            <Link
              to="/create"
              className={`flex items-center space-x-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                isActive('/create')
                  ? 'bg-primary-100 text-primary-700'
                  : 'bg-primary-600 text-white hover:bg-primary-700'
              }`}
            >
              <Plus className="w-4 h-4" />
              <span>Post Item</span>
            </Link>

            {/* Auth */}
            {user ? (
              <div className="flex items-center space-x-2">
                <span className="hidden sm:flex items-center space-x-1 text-sm text-gray-700">
                  <User className="w-4 h-4 text-gray-400" />
                  <span>{user.name}</span>
                </span>
                <button
                  onClick={handleLogout}
                  title="Sign out"
                  className="flex items-center space-x-1 px-3 py-2 rounded-md text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  <span className="hidden sm:inline">Sign out</span>
                </button>
              </div>
            ) : GOOGLE_CLIENT_ID ? (
              <div ref={googleButtonRef} />
            ) : (
              <button
                onClick={handleDevLogin}
                className="px-3 py-2 rounded-md text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 transition-colors"
              >
                Sign in
              </button>
            )}
          </nav>
        </div>
      </div>
    </header>
  );
};

export default Header;
