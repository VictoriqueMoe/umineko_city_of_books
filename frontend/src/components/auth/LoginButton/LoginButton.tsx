import { useLocation, useNavigate } from "react-router";
import { Button } from "../../Button/Button";

export function LoginButton() {
    const navigate = useNavigate();
    const originalLocation = useLocation();

    return (
        <Button 
            variant="primary" 
            onClick={() => {
                navigate("/login", { state: { from: originalLocation } })
            }}
        >
            Sign In
        </Button>
    );
}
