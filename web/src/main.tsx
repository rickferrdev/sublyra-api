import { StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { ArrowRight, CheckIcon } from "lucide-react";

const element = document.getElementById("root");
if (!element) {
  throw new Error("Element root not found");
}

function SubscriptionPage() {
  const [selectedPlan, setSelectedPlan] = useState<string>("pro");
  const [email, setEmail] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const plans = [
    {
      id: "basic",
      name: "Basic",
      price: "Free",
      period: "forever",
      description: "For basic personal use and local testing.",
      features: ["Standard API access", "100 requests / day", "Community support"],
    },
    {
      id: "pro",
      name: "Pro",
      price: "$20",
      period: "month",
      description: "Speed and features for hardcore developers.",
      features: [
        "Everything in Basic",
        "Unlimited requests",
        "Access to advanced models",
        "Execution queue priority",
        "Priority email support"
      ],
      isPopular: true,
    },
    {
      id: "business",
      name: "Business",
      price: "$99",
      period: "month",
      description: "For teams needing scale and enterprise security.",
      features: [
        "Everything in Pro",
        "Admin dashboard",
        "99.9% SLA guaranteed",
        "SSO integration"
      ],
    },
  ];

  const handleSubmit = async () => {
    if (!email) {
      setError("Please enter an email address");
      return;
    }

    setIsLoading(true);
    setError(null);
    setSuccessMessage(null);

    const planName = plans.find((p) => p.id === selectedPlan)?.name;

    try {
      const response = await fetch("http://localhost:8080/api/v1/subscription", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ email, name: planName }),
      });

      let data;
      const contentType = response.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        data = await response.json();
      } else {
        // Fallback for non-JSON responses (e.g. native rate limit, gateway timeouts)
        if (response.status === 429) {
          data = { code: "TOO_MANY_REQUESTS", message: "Too many requests (Rate Limit)." };
        } else {
          data = { code: "INTERNAL_ERROR", message: `Server error (${response.status})` };
        }
      }

      if (!response.ok) {
        let errorMessage = data.message || "Unknown error";

        // Mapping Go error codes (ports/error.go) to user-friendly messages
        switch (data.code) {
          case "INTERNAL_ERROR":
          case "SERVICE_UNAVAILABLE":
            errorMessage = "Server is down or encountered an internal error.";
            break;
          case "UNAUTHORIZED":
          case "UNAUTHENTICATED":
            errorMessage = "Unauthorized access.";
            break;
          case "TOO_MANY_REQUESTS":
            errorMessage = "Too many requests. Please try again later.";
            break;
          case "CONFLICT":
            errorMessage = "This subscription already exists or is pending.";
            break;
          case "INVALID_INPUT":
          case "VALIDATION_ERROR":
            errorMessage = "Invalid data provided.";
            break;
        }

        console.error(`[API ERROR] Status: ${response.status} | Code: ${data.code} | Message: ${data.message}`);
        alert(errorMessage);
        throw new Error(errorMessage);
      }

      setSuccessMessage("Subscription started! Please check your email.");
      setEmail("");
      alert("Success! Please check your email to confirm your subscription.");
    } catch (err: any) {
      let finalError = err.message || "An unexpected error occurred.";

      // Fallback for when the server is completely unreachable (CORS, offline, no network)
      if (finalError === "Failed to fetch") {
        finalError = "Server is unreachable or no internet connection.";
        console.error(`[NETWORK ERROR] Connection refused or server unreachable.`);
        alert(finalError);
      }

      setError(finalError);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-[#ededed] font-sans selection:bg-neutral-800 flex items-center justify-center p-6">
      <div className="max-w-4xl w-full mx-auto">

        <div className="mb-12 text-center">
          <h1 className="text-3xl font-medium tracking-tight mb-3 text-[#ededed]">Select a Plan</h1>
          <p className="text-[#a1a1aa] text-sm">Upgrade or downgrade at any time.</p>
        </div>

        {/* Plan Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {plans.map((plan) => (
            <div
              key={plan.id}
              onClick={() => !isLoading && setSelectedPlan(plan.id)}
              className={`relative flex flex-col p-6 rounded-xl cursor-pointer transition-all duration-200 border ${selectedPlan === plan.id
                ? "border-[#ededed] bg-[#121212]"
                : "border-[#27272a] bg-[#0a0a0a] hover:bg-[#121212] hover:border-[#3f3f46]"
                } ${isLoading ? "opacity-50 pointer-events-none" : ""}`}
            >
              {plan.isPopular && (
                <div className="absolute -top-2.5 left-1/2 -translate-x-1/2 bg-[#ededed] text-[#0a0a0a] text-[10px] font-semibold px-2 py-0.5 rounded-full uppercase tracking-widest">
                  Recommended
                </div>
              )}

              <div className="mb-5">
                <h3 className="text-sm font-medium text-[#ededed] mb-1">{plan.name}</h3>
                <p className="text-xs text-[#a1a1aa] h-8 leading-relaxed">{plan.description}</p>
              </div>

              <div className="mb-6 flex items-baseline gap-1">
                <span className="text-3xl font-medium text-[#ededed] tracking-tight">{plan.price}</span>
                {plan.period && <span className="text-xs text-[#a1a1aa] font-medium">{plan.period}</span>}
              </div>

              <div className="flex-1">
                <ul className="space-y-3">
                  {plan.features.map((feature, idx) => (
                    <li key={idx} className="flex items-start gap-2.5 text-xs text-[#a1a1aa]">
                      <CheckIcon className="w-3.5 h-3.5 text-[#ededed] shrink-0 mt-0.5" />
                      <span className="leading-relaxed">{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          ))}
        </div>

        {/* Action Button */}
        <div className="mt-12 flex flex-col items-center gap-4">
          <input
            type="email"
            placeholder="Enter your email"
            className="bg-[#121212] border border-[#27272a] focus:border-[#ededed] outline-none text-[#ededed] px-4 py-2 rounded text-sm w-full max-w-sm transition-colors placeholder:text-[#52525b]"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={isLoading}
          />
          {error && <p className="text-red-400 text-sm">{error}</p>}
          {successMessage && <p className="text-green-400 text-sm">{successMessage}</p>}
          <button
            onClick={handleSubmit}
            disabled={isLoading || !email}
            className="bg-[#ededed] text-[#0a0a0a] hover:bg-[#d4d4d8] disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium text-sm px-10 py-3 rounded flex items-center gap-2"
          >
            {isLoading ? "Processing..." : `Continue with ${plans.find((p) => p.id === selectedPlan)?.name}`}
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>

      </div>
    </div>
  );
}

createRoot(element).render(
  <StrictMode>
    <SubscriptionPage />
  </StrictMode>
);
