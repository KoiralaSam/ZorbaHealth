export default function VerifyEmailPage() {
  return (
    <main className="min-h-screen bg-gradient-to-b from-blue-50 to-white">
      <div className="flex flex-col items-center justify-center min-h-screen gap-6 px-4 py-8">
        <div className="bg-white p-8 rounded-2xl shadow-lg max-w-md w-full text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            Check your email
          </h2>
          <p className="text-gray-600 text-sm mb-4">
            We&apos;ve sent a verification link to your email address. Please
            open the email and click the link to activate your account.
          </p>
          <p className="text-gray-500 text-xs">
            If you don&apos;t see the email, check your spam or junk folder.
          </p>
        </div>
      </div>
    </main>
  );
}
